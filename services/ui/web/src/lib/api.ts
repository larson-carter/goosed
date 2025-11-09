const metaBaseURL = (() => {
  if (typeof document === "undefined") {
    return "";
  }
  const meta = document.querySelector('meta[name="goosed:api-base"]');
  if (meta instanceof HTMLMetaElement) {
    const content = meta.content.trim();
    if (content) {
      return content.replace(/\/$/, "");
    }
  }
  return "";
})();

const API_BASE_URL = (() => {
  const raw = import.meta.env.VITE_API_BASE_URL;
  if (typeof raw === "string" && raw.trim().length > 0) {
    return raw.trim().replace(/\/$/, "");
  }
  if (metaBaseURL) {
    return metaBaseURL;
  }
  if (import.meta.env.DEV) {
    return "/api";
  }
  return "";
})();

export function apiURL(path: string) {
  if (!path.startsWith("/")) {
    throw new Error(
      `apiURL expects an absolute path that begins with '/' but received: ${path}`,
    );
  }
  if (!API_BASE_URL) {
    return path;
  }
  return `${API_BASE_URL}${path}`;
}

export function getAPIBaseURL() {
  return API_BASE_URL;
}
