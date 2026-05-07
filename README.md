# HomePlan

A static GitHub Pages dashboard for tracking home repairs room by room. The dashboard is intentionally simple: one `index.html` file with plain HTML, CSS, and JavaScript, no build step, and no external dependencies.

## Local use

Open `index.html` directly in a browser.

## GitHub Pages setup

1. Merge the init branch into `main`.
2. Open the repository on GitHub.
3. Go to **Settings** -> **Pages**.
4. Under **Build and deployment**, set **Source** to **Deploy from a branch**.
5. Select the `main` branch and the `/root` folder.
6. Save and wait for GitHub to publish the Pages URL.

## Notes

- The dashboard is privacy-safe and uses room names only.
- Project data currently powers the page from a JavaScript object inside `index.html`.
- A matching `tasks.json` file is included so future tooling can mark tasks complete or update completion dates.
