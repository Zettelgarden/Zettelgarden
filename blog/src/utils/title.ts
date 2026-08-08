const zettel_env = import.meta.env.VITE_ENV || "";

// Base title is mutable: SettingsContext applies the instance's site_name
// (config.yaml) once loaded, so a self-hosted instance brands its own tabs.
let baseTitle = "Zettelgarden" + zettel_env;

/**
 * Sets the base title suffix used by setDocumentTitle (e.g. the instance
 * site_name from runtime settings).
 */
export function setBaseTitle(title: string) {
  baseTitle = (title || "Zettelgarden") + zettel_env;
}

/**
 * Sets the document title in a standardized format across the app.
 * @param pageTitle Optional page-specific title. If not provided, only the base title will be used.
 */
export function setDocumentTitle(pageTitle?: string) {
  document.title = pageTitle ? `${pageTitle} - ${baseTitle}` : baseTitle;
}
