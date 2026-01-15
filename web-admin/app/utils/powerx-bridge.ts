export const PLUGIN_ID = "com.powerx.plugin.scrm";
export const PLUGIN_ADMIN_PREFIX = `/_p/${PLUGIN_ID}/admin`;
const PLUGIN_SLUG = PLUGIN_ID.split(".").pop() || PLUGIN_ID;
const LEGACY_EMBEDDED_PREFIX = `${PLUGIN_ADMIN_PREFIX}/plugins/${PLUGIN_SLUG}`;
const LEGACY_LOCAL_PREFIX = `/plugins/${PLUGIN_SLUG}`;

const SUPPORTED_LOCALES = ["zh", "en"] as const;
type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

const stripEmptySegments = (segments: string[]) =>
  segments.filter((segment) => segment.trim().length > 0);

export const ensureLeadingSlash = (value: string) =>
  value.startsWith("/") ? value : `/${value}`;

export const normalizePath = (value: string): string => {
  if (!value) {
    return "/";
  }

  const withLeadingSlash = ensureLeadingSlash(value.trim());
  const segments = stripEmptySegments(withLeadingSlash.split("/"));

  if (!segments.length) {
    return "/";
  }

  return `/${segments.join("/")}`;
};

// 需要把语言段插入到插件管理路径内部
const EMBEDDED_PREFIXES = [
  PLUGIN_ADMIN_PREFIX,
  LEGACY_EMBEDDED_PREFIX,
].map((prefix) => normalizePath(prefix));

const maybeBuildEmbeddedLocalePath = (
  localePrefix: string,
  normalizedPath: string
): string | null => {
  if (!localePrefix) {
    return null;
  }

  const locale = localePrefix.replace(/^\/+/, "").trim();
  if (!locale) {
    return null;
  }

  for (const prefix of EMBEDDED_PREFIXES) {
    if (normalizedPath === prefix || normalizedPath.startsWith(`${prefix}/`)) {
      const suffix = normalizedPath.slice(prefix.length);
      const normalizedSuffix = suffix
        ? suffix.startsWith("/")
          ? suffix
          : `/${suffix}`
        : "";
      if (
        normalizedSuffix === `/${locale}` ||
        normalizedSuffix.startsWith(`/${locale}/`)
      ) {
        return normalizedPath;
      }
      return normalizePath(`${prefix}/${locale}${normalizedSuffix}`);
    }
  }

  return null;
};

export const joinLocaleWithPath = (
  localePrefix: string,
  path: string
): string => {
  const normalized = normalizePath(path || "/");
  if (!localePrefix) {
    return normalized;
  }

  if (normalized === "/") {
    return localePrefix || "/";
  }

  const embeddedLocalePath = maybeBuildEmbeddedLocalePath(localePrefix, normalized);
  if (embeddedLocalePath) {
    return embeddedLocalePath;
  }

  return `${localePrefix}${normalized}`;
};

export interface LocaleStripResult {
  localePrefix: string;
  pathWithoutLocale: string;
}

export const stripLocalePrefix = (path: string): LocaleStripResult => {
  const normalizedPath = normalizePath(path || "/");

  const segments = stripEmptySegments(normalizedPath.split("/"));
  const potentialLocale = segments[0];

  if (
    potentialLocale &&
    SUPPORTED_LOCALES.includes(potentialLocale as SupportedLocale)
  ) {
    const remainingSegments = segments.slice(1);
    const remainingPath = remainingSegments.length
      ? `/${remainingSegments.join("/")}`
      : "/";

    return {
      localePrefix: `/${potentialLocale}`,
      pathWithoutLocale: remainingPath,
    };
  }

  return {
    localePrefix: "",
    pathWithoutLocale: normalizedPath,
  };
};

type RouteContext = "embedded" | "local";

export interface ExtractedPluginRoute {
  localePrefix: string;
  internalPath: string;
  rawPath: string;
  context: RouteContext;
  legacy: boolean;
}

const buildExtractedRoute = (
  localePrefix: string,
  rawPath: string,
  internal: string,
  context: RouteContext,
  legacy: boolean
): ExtractedPluginRoute => ({
  localePrefix,
  rawPath,
  internalPath: normalizePath(internal || "/"),
  context,
  legacy,
});

export const extractInternalRoute = (
  fullPath: string
): ExtractedPluginRoute | null => {
  const { localePrefix, pathWithoutLocale } = stripLocalePrefix(fullPath);
  const normalizedWithoutLocale = normalizePath(pathWithoutLocale);

  if (normalizedWithoutLocale.startsWith(PLUGIN_ADMIN_PREFIX)) {
    const internal = normalizedWithoutLocale.slice(PLUGIN_ADMIN_PREFIX.length);
    return buildExtractedRoute(
      localePrefix,
      normalizedWithoutLocale,
      internal,
      "embedded",
      false
    );
  }

  if (normalizedWithoutLocale.startsWith(LEGACY_EMBEDDED_PREFIX)) {
    const internal = normalizedWithoutLocale.slice(LEGACY_EMBEDDED_PREFIX.length);
    return buildExtractedRoute(
      localePrefix,
      normalizedWithoutLocale,
      internal,
      "embedded",
      true
    );
  }

  if (normalizedWithoutLocale.startsWith(LEGACY_LOCAL_PREFIX)) {
    const internal = normalizedWithoutLocale.slice(LEGACY_LOCAL_PREFIX.length);
    return buildExtractedRoute(
      localePrefix,
      normalizedWithoutLocale,
      internal,
      "local",
      true
    );
  }

  return null;
};

export const isPluginAdminPath = (fullPath: string) => {
  const extracted = extractInternalRoute(fullPath);
  return Boolean(extracted && extracted.context === "embedded");
};

export const resolveCanonicalInternalPath = (
  internalPath: string
): string => {
  const normalized = normalizePath(internalPath);
  const aliasTarget = resolveInternalAlias(normalized);
  if (!aliasTarget) {
    return normalized;
  }

  return normalizePath(aliasTarget);
};

const stripLegacyPluginPrefix = (path: string): string => {
  if (!path.startsWith(`/plugins/${PLUGIN_SLUG}`)) {
    return path;
  }

  const stripped = path.slice(`/plugins/${PLUGIN_SLUG}`.length);
  return normalizePath(stripped || "/");
};

export const resolveInternalAlias = (internalPath: string): string | null => {
  const normalized = normalizePath(internalPath);

  const sanitizedPath = stripLegacyPluginPrefix(normalized);

  if (sanitizedPath !== normalized) {
    return resolveInternalAlias(sanitizedPath);
  }

  if (sanitizedPath === "/" || sanitizedPath === "/dashboard") {
    return "/intro";
  }

  if (sanitizedPath === "/intro") {
    return "/intro";
  }

  if (sanitizedPath.startsWith("/templates")) {
    return sanitizedPath;
  }

  return null;
};

export interface NavigationTarget {
  localePrefix: string;
  internalPath: string;
  targetPath: string;
  finalPath: string;
}

export const buildNavigationTarget = (
  fullPath: string
): NavigationTarget | null => {
  const extracted = extractInternalRoute(fullPath);
  if (!extracted) {
    return null;
  }

  const normalizedFullPath = normalizePath(fullPath);
  const aliasTarget = resolveInternalAlias(extracted.internalPath);
  const targetInternal = normalizePath(aliasTarget || extracted.internalPath);

  let pathWithinLocale: string;
  if (extracted.context === "embedded") {
    const baseTarget =
      targetInternal === "/"
        ? `${PLUGIN_ADMIN_PREFIX}/`
        : `${PLUGIN_ADMIN_PREFIX}${targetInternal}`;
    pathWithinLocale = normalizePath(baseTarget);
  } else {
    pathWithinLocale = targetInternal;
  }

  const finalPath = joinLocaleWithPath(
    extracted.localePrefix,
    pathWithinLocale
  );
  const normalizedFinalPath = normalizePath(finalPath);

  if (normalizedFinalPath === normalizedFullPath) {
    return null;
  }

  return {
    localePrefix: extracted.localePrefix,
    internalPath: extracted.internalPath,
    targetPath: targetInternal,
    finalPath: normalizedFinalPath,
  };
};
