# Playbook 05 - Skills (AgentSkills)

Goal: load skills from disk and use progressive disclosure.

Incremental reuse:

- Add `internal/skills` to load skills and derive allowlists.

What to implement:

- Create `skills/<skill-name>/SKILL.md` with frontmatter:
  - `name`, `description`, `license`, `compatibility`, `allowed-tools`
- Ensure the directory name matches the skill name (lowercase + dashes).
- Add optional resources in `skills/<skill-name>/references|assets|scripts`.
- Load skills with `agent.WithSkillsFromDir`.
- Build a tool allowlist using `governance.AllowlistFromSkills`.
- Accept a user prompt that requires the skill to answer.
- Ensure the agent reports that it used the skill (log, event, or response text).
- Reuse provider/config wiring from step 02 via shared helpers.

Suggested checks:

- Calling the skill tool returns instructions and resource list.
- `load_resource` returns a file from the skill directory.
- A prompt that needs the skill triggers the skill tool.

Manual tests:

- "Use the `<skill-name>` skill to answer this question."

Expected behavior:

- The skill tool is invoked (progressive disclosure).
- The agent states that it used the skill.

Checklist:

- [ ] Skill name matches directory name.
- [ ] Skill tool lists resources.
- [ ] Allowed tools map into the tool filter.

References:

- `examples/04-skills-agent`
- `pkg/skills`
- `pkg/governance`
