# HomePlan

A static GitHub Pages dashboard for tracking home repairs room by room with no external dependencies.

## Local use

Run the static site through Podman/Nginx from the repository root, then open the local URL. The dashboard loads task data from `tasks.json`, so opening `index.html` directly from disk may be blocked by the browser.

```powershell
.\scripts\serve.ps1
```

Then visit `http://localhost:8080`.

## Task data

`tasks.json` is the source of truth for task titles, status, progress, subtasks, dates, and items needed. `index.html` only stores the floor-plan layout and rendering code.

## GitHub Pages

Visit [github page](https://jasonbuchanan145.github.io/HomePlan/)
