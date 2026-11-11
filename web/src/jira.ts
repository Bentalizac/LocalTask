// Splits free-form input ("proj-12, PROJ-34 PROJ-56") into unique uppercase Jira
// keys, in the order first written.
export function parseJiraKeys(text: string): string[] {
  const keys = text
    .split(/[\s,]+/)
    .map((k) => k.trim().toUpperCase())
    .filter(Boolean)
  return [...new Set(keys)]
}
