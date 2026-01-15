import { existsSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { resolve } from 'node:path';

const installScript = resolve('node_modules', 'lightningcss', 'install.js');

if (existsSync(installScript)) {
  try {
    await import(pathToFileURL(installScript));
  } catch (error) {
    console.warn('[postinstall] lightningcss install.js 执行失败，将继续安装流程');
    console.warn(error);
  }
} else {
  console.info('[postinstall] 未检测到 lightningcss install.js，跳过该步骤');
}
