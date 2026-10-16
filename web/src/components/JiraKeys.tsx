interface Props {
  keys?: string[]
  baseUrl?: string
}

// Renders a ticket's Jira issue keys as chips, each linking to
// <baseUrl>/browse/<key> when a Jira site is configured.
export function JiraKeys({ keys, baseUrl }: Props) {
  return (
    <>
      {(keys ?? []).map((key) =>
        baseUrl ? (
          <a
            className="tag jira-tag"
            key={key}
            href={`${baseUrl}/browse/${encodeURIComponent(key)}`}
            target="_blank"
            rel="noreferrer"
            draggable={false}
            onClick={(e) => e.stopPropagation()}
            title={`Open ${key} in Jira`}
          >
            {key}
          </a>
        ) : (
          <span className="tag jira-tag" key={key} title="Set jira_base_url in taskboard.config.yaml to link this key">
            {key}
          </span>
        ),
      )}
    </>
  )
}
