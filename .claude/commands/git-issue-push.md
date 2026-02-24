Create a GitHub issue in `petar-djukic/cobbler-scaffold`.

## Input

$ARGUMENTS

## Steps

1. Determine type from the input: keywords like "bug", "fix", "broken", "crash" → bug; otherwise → enhancement.
2. Draft a concise title and well-structured body.
   - **Bug**: problem, expected vs actual behavior, reproduction steps if provided.
   - **Enhancement**: description and acceptance criteria.
3. Create the issue:
   ```
   gh issue create --repo petar-djukic/cobbler-scaffold --title "<title>" --body "<body>" --label "<bug|enhancement>"
   ```
4. Report the issue URL.
