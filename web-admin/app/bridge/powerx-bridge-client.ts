// powerx-bridge-client.ts
// Drop-in 插件侧 Bridge 客户端（带调试日志）
// 仅用于 iframe 内（插件页）
// ------------------------------------------------------------

export type ThemeKey = 'light' | 'dark' | 'system'

export interface PowerXSyncPayload {
  source: 'powerx'
  type: 'sync'
  locale: string
  theme: ThemeKey
  hostOrigin?: string
  pluginId?: string
  instanceId?: string
}

export interface PowerXThemePayload {
  source: 'powerx'
  type: 'theme'
  theme: ThemeKey
}

export interface PowerXLocalePayload {
  source: 'powerx'
  type: 'locale'
  locale: string
}

export interface PowerXAuthTokenPayload {
  source: 'powerx'
  type: 'auth-token'
  accessToken: string
  refreshToken?: string
  tokenType?: string
  expiresIn?: number
  expiresAt?: number
  scope?: string
  pluginId?: string
  hostOrigin?: string
}

export type PowerXMessage =
  | PowerXSyncPayload
  | PowerXThemePayload
  | PowerXLocalePayload
  | PowerXAuthTokenPayload

export interface PluginReadyPayload {
  source: 'plugin'
  type: 'ready'
  pluginId?: string
  instanceId?: string
}

export interface PluginRequestSyncPayload {
  source: 'plugin'
  type: 'request-sync'
}

export interface PluginPingPayload {
  source: 'plugin'
  type: 'ping'
  ts: number
}

export type PluginToHost = PluginReadyPayload | PluginRequestSyncPayload | PluginPingPayload

export interface BridgeOptions {
  pluginId?: string
  instanceId?: string
  debug?: boolean
  /** 允许的宿主来源列表（精确 origin，如 http://127.0.0.1:3036） */
  allowedOrigins?: string[]
  /** 回调：接到主题变化 */
  onTheme?: (theme: ThemeKey) => void
  /** 回调：接到语言变化 */
  onLocale?: (locale: string) => void
  /** 回调：接到同步包（包含 locale、theme、hostOrigin 等） */
  onSync?: (payload: PowerXSyncPayload) => void
  /** 回调：接到宿主注入的短时 token */
  onAuthToken?: (payload: PowerXAuthTokenPayload) => void
}

export class PowerXBridgeClient {
  public pluginId?: string
  public instanceId?: string
  public debug: boolean
  public onTheme?: (t: ThemeKey) => void
  public onLocale?: (l: string) => void
  public onSync?: (p: PowerXSyncPayload) => void
  public onAuthToken?: (p: PowerXAuthTokenPayload) => void

  private allowedOrigins: Set<string>
  private _bound = false
  private _stopped = false
  private _lastHostOrigin?: string

  constructor(opts: BridgeOptions = {}) {
    this.pluginId = opts.pluginId
    this.instanceId = opts.instanceId
    this.debug = !!opts.debug
    this.onTheme = opts.onTheme
    this.onLocale = opts.onLocale
    this.onSync = opts.onSync
    this.onAuthToken = opts.onAuthToken

    const defaultAllow = this._defaultAllowedOrigins()
    const extra = (opts.allowedOrigins || []).filter(Boolean)
    // 放宽：统一加入 '*'，避免宿主代理导致的 origin 不一致
    this.allowedOrigins = new Set(['*', ...defaultAllow, ...extra])

    this._log('init', {
      pluginId: this.pluginId,
      instanceId: this.instanceId,
      allowedOrigins: Array.from(this.allowedOrigins),
      location: window.location.origin,
      referrer: document.referrer,
      mode: import.meta.env.MODE
    })
  }

  /** 启动监听（幂等） */
  start() {
    this._log('start() called, binding message listener')
    if (this._bound || this._stopped) return
    this._bound = true

    // 原始消息“回显”（仅调试用）
    window.addEventListener(
      'message',
      (e) => {
        if (!this.debug) return
        const data = (e as MessageEvent<any>).data
        if (!data || (data.source !== 'powerx' && data.source !== 'plugin')) return
        const brief =
          data?.type === 'sync'
            ? { type: data.type, locale: data.locale, theme: data.theme }
            : data?.type === 'locale'
              ? { type: data.type, locale: data.locale }
              : data?.type === 'theme'
                ? { type: data.type, theme: data.theme }
                : { type: data?.type }
        this._log('raw message', { origin: (e as MessageEvent).origin, data: brief })
      },
      false
    )

    // 业务监听
    window.addEventListener('message', this._handle, false)

    // 启动即汇报一次 ready
    this.ready()
  }

  /** 停止监听 */
  stop() {
    if (!this._bound) return
    this._stopped = true
    this._bound = false
    window.removeEventListener('message', this._handle, false)
    this._log('stopped')
  }

  /** 告知宿主：插件就绪 */
  ready() {
    const payload: PluginReadyPayload = {
      source: 'plugin',
      type: 'ready',
      pluginId: this.pluginId,
      instanceId: this.instanceId
    }
    this._sendToParent(payload)
  }

  /** 主动请求一次同步（让宿主回发 sync） */
  requestSync() {
    const payload: PluginRequestSyncPayload = { source: 'plugin', type: 'request-sync' }
    this._sendToParent(payload)
  }

  /** 发送心跳（调试用） */
  ping() {
    const payload: PluginPingPayload = { source: 'plugin', type: 'ping', ts: Date.now() }
    this._sendToParent(payload)
  }

  // ----------------- 内部实现 -----------------

  private _handle = (e: MessageEvent<any>) => {
    const data = e.data as PowerXMessage
    if (!data || data.source !== 'powerx') return

    // 校验 origin
    if (!this._isAllowedOrigin(e.origin)) {
      this._log('drop message: origin not allowed', { origin: e.origin, type: data.type })
      return
    }

    switch (data.type) {
      case 'sync': {
        // 记住宿主 origin，便于之后上行对齐
        this._lastHostOrigin = data.hostOrigin || e.origin
        this._log('onSync <-', {
          locale: data.locale,
          theme: data.theme,
          hostOrigin: data.hostOrigin || e.origin
        })
        try {
          this.onSync?.(data)
        } catch (err) {
          this._log('onSync error', err)
        }
        break
      }
      case 'locale': {
        this._log('onLocale <-', data.locale)
        try {
          this.onLocale?.(data.locale)
        } catch (err) {
          this._log('onLocale error', err)
        }
        break
      }
      case 'theme': {
        this._log('onTheme  <-', data.theme)
        try {
          this.onTheme?.(data.theme)
        } catch (err) {
          this._log('onTheme error', err)
        }
        break
      }
      case 'auth-token': {
        this._lastHostOrigin = data.hostOrigin || e.origin
        if (!data.accessToken) {
          this._log('drop auth-token: empty accessToken', data)
          return
        }
        this._log('onAuthToken <-', {
          origin: e.origin,
          expiresIn: data.expiresIn,
          expiresAt: data.expiresAt,
          pluginId: data.pluginId,
          token: data.accessToken ? `${data.accessToken.slice(0, 6)}...` : '<empty>'
        })
        try {
          this.onAuthToken?.(data)
        } catch (err) {
          this._log('onAuthToken error', err)
        }
        break
      }
    }
  }

  private _sendToParent(payload: PluginToHost) {
    const target = window.parent || window.top
    if (!target || target === window) {
      this._log('no parent window; skip postMessage', payload)
      return
    }
    // 调试打印：我们把“目标”尽量标注出来
    this._log('-> postMessage', {
      to: 'parent',
      targetOrigin: this._lastHostOrigin || '(unknown until sync)',
      payload
    })
    try {
      // 安全策略：在未收到 sync 之前，无法准确知道 host 的 origin，这里用 '*' 纯调试；
      // 一旦收到 sync（带 hostOrigin），你可以替换为那个精确 origin，进一步收紧。
      target.postMessage(payload, this._lastHostOrigin || '*')
    } catch (err) {
      this._log('postMessage error', err)
    }
  }

  private _isAllowedOrigin(origin: string): boolean {
    // 如果开发者显式传了 '*'，直接放行（仅调试场景）
    if (this.allowedOrigins.has('*')) return true
    // 允许 list 中的 origin
    if (this.allowedOrigins.has(origin)) return true
    // 某些浏览器在 file/特殊协议下会给空字符串，这里保守拒绝
    return false
  }

  private _defaultAllowedOrigins(): string[] {
    const res: string[] = []
    // 1) 来自 referrer 的 origin（大多数 iframe 都会带）
    try {
      if (document.referrer) res.push(new URL(document.referrer).origin)
    } catch {}
    // 2) 若插件与宿主同域嵌入，允许自身
    try {
      res.push(window.location.origin)
    } catch {}
    // 3) 开发环境自动放宽
    if (import.meta.env.DEV) res.push('*')
    return Array.from(new Set(res.filter(Boolean)))
  }

  private _log(...args: any[]) {
    if (this.debug) {
      console.info('[DBG][Plugin]', ...args)
    }
  }
}

// ------------------------------------------------------------
// 工具方法：初始化并启动（常用入口）
// ------------------------------------------------------------
export function initPowerXBridge(opts: BridgeOptions = {}) {
  // 单例防重复（HMR 场景）
  const k = '__POWERX_BRIDGE__'
  const g = window as any
  if (g[k]) {
    if (opts.debug) console.info('[DBG][Plugin] reuse existing bridge instance')
    const client = g[k] as PowerXBridgeClient
    // 更新回调/配置（可选）
    client.onLocale = opts.onLocale || client.onLocale
    client.onTheme = opts.onTheme || client.onTheme
    client.onSync = opts.onSync || client.onSync
    client.onAuthToken = opts.onAuthToken || client.onAuthToken
    return client
  }

  const client = new PowerXBridgeClient(opts)
  client.start()
  g[k] = client

  if (opts.debug) {
    // 暴露一些调试入口
    g.__PX_DEBUG__ = {
      info: () => ({
        location: window.location.origin,
        referrer: document.referrer,
        lastHostOrigin: (client as any)._lastHostOrigin,
        allowedOrigins: 'hidden in client.allowedOrigins (Set)',
        pluginId: client.pluginId,
        instanceId: client.instanceId
      }),
      ready: () => client.ready(),
      sync: () => client.requestSync(),
      ping: () => client.ping()
    }
    console.info('[DBG][Plugin] debug helpers available: __PX_DEBUG__')
  }

  return client
}
