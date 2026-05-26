import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

async function resetHouse(request: APIRequestContext, page: Page) {
  await request.delete("/api/dev/users/user-1/house/current").catch(() => undefined);
  await page.goto("/");
  await page.evaluate(() => window.localStorage.clear());
}

async function openFresh(page: Page) {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Plan home projects by room, not just by list." })).toBeVisible();
}

async function createBlankHouse(page: Page) {
  await page.getByRole("button", { name: "Create Blank House" }).click();
  await expect(page.getByRole("heading", { name: "Interactive Floor Plan" })).toBeVisible();
}

async function createTaskFromRoomPanel(page: Page, title: string) {
  await page.getByRole("button", { name: "Add Task" }).click();
  const dialog = page.getByRole("dialog", { name: "Add Task" });
  await expect(dialog).toBeVisible();
  await dialog.getByLabel("Title").fill(title);
  await dialog.getByRole("button", { name: "Create Task" }).click();
  await expect(dialog).toBeHidden();
}

function taskInput(page: Page, title: string) {
  return page.locator(`input[value="${title}"]`);
}

test.beforeEach(async ({ request, page }) => {
  await resetHouse(request, page);
});

test("empty state shows onboarding splash and creates a blank house", async ({ page }) => {
  await openFresh(page);
  await expect(page.getByRole("group", { name: "Static example of a room-by-room home project plan" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Kitchen • 3 open tasks" })).toBeVisible();
  await expect(page.getByText("Next: book electrician")).toBeVisible();
  await expect(page.getByText("Saving uses an essential session cookie")).toBeVisible();
  await expect(page.getByRole("link", { name: "Privacy Policy" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Cookie Policy" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Start with projects" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Start with rooms" })).toBeVisible();
  await expect(page.getByText("Project summary")).toHaveCount(0);
  await expect(page.getByText("Critical")).toHaveCount(0);

  await createBlankHouse(page);

  await expect(page.getByLabel("Floor Name")).toHaveValue("Main Floor");
  await expect(page.getByLabel("Room Name")).toHaveValue("Main Room");
});

test("privacy and cookie policy pages render", async ({ page }) => {
  await page.goto("/privacy");
  await expect(page.getByRole("heading", { name: "Privacy Policy" })).toBeVisible();
  await expect(page.getByText("Google account ID")).toBeVisible();
  await expect(page.getByRole("link", { name: "Cookie Policy" })).toBeVisible();

  await page.goto("/cookies");
  await expect(page.getByRole("heading", { name: "Cookie Policy" })).toBeVisible();
  await expect(page.getByText("homeplan_session")).toBeVisible();
  await expect(page.getByText("no analytics, advertising, cross-site tracking")).toBeVisible();
});

test("project setup creates rooms and starter tasks", async ({ page }) => {
  await openFresh(page);

  await page.getByRole("button", { name: "Start with projects" }).click();
  await expect(page.getByRole("button", { name: "Add More Areas Or Rooms" })).toBeVisible();
  await page.getByLabel("House Name").fill("Lake House");
  await page.getByLabel("Task 1").fill("Fix cabinet hinge");
  await page.getByRole("button", { name: "Create House From Projects" }).click();

  await expect(page.getByLabel("Floor Name")).toHaveValue("Main Floor");
  await expect(page.getByLabel("Room Name")).toHaveValue("Kitchen");
  await expect(taskInput(page, "Fix cabinet hinge")).toBeVisible();
});

test("project setup can place work on the exterior area", async ({ page }) => {
  await openFresh(page);

  await page.getByRole("button", { name: "Start with projects" }).click();
  await page.getByLabel("Floor Or Area").selectOption("exterior");
  await page.getByLabel("Area Or Room").selectOption("Exterior");
  await page.getByLabel("Task 1").fill("Clean gutters");
  await page.getByRole("button", { name: "Create House From Projects" }).click();

  await expect(page.getByRole("tab", { name: "Exterior" })).toBeVisible();
  await expect(page.getByLabel("Floor Name")).toHaveValue("Exterior");
  await expect(page.getByLabel("Room Name")).toHaveValue("Exterior");
  await expect(taskInput(page, "Clean gutters")).toBeVisible();
});

test("room setup creates multiple floors and rooms", async ({ page }) => {
  await openFresh(page);

  await page.getByRole("button", { name: "Start with rooms" }).click();
  await page.getByLabel("House Name").fill("Lake House");
  await page.getByRole("button", { name: "Create House From Rooms" }).click();

  await expect(page.getByRole("tab", { name: "Main Floor" })).toBeVisible();
  await expect(page.getByRole("tab", { name: "Second Floor" })).toBeVisible();
  await expect(page.getByLabel("Room Name")).toHaveValue("Kitchen");
  await page.getByRole("tab", { name: "Second Floor" }).click();
  await expect(page.getByLabel("Room Name")).toHaveValue("Bedroom");
});

test("room panel guided task form accepts title-only submit", async ({ page }) => {
  await openFresh(page);
  await createBlankHouse(page);

  await createTaskFromRoomPanel(page, "Replace light switch");

  await expect(taskInput(page, "Replace light switch")).toBeVisible();
});

test("all tasks guided task form requires a target", async ({ page }) => {
  await openFresh(page);
  await createBlankHouse(page);
  await page.getByRole("button", { name: "List All Tasks" }).click();
  await page.getByRole("button", { name: "Add Task" }).click();

  const dialog = page.getByRole("dialog", { name: "Add Task" });
  await dialog.getByLabel("Title").fill("Patch drywall");
  await expect(dialog.getByRole("button", { name: "Create Task" })).toBeDisabled();
  await dialog.getByLabel("Target").selectOption({ label: "Main Floor / Main Room" });
  await dialog.getByRole("button", { name: "Create Task" }).click();

  await expect(taskInput(page, "Patch drywall")).toBeVisible();
});

test("all tasks form can create an unplaced room with a task", async ({ page }) => {
  await openFresh(page);
  await createBlankHouse(page);
  await page.getByRole("button", { name: "List All Tasks" }).click();
  await page.getByRole("button", { name: "Add Task" }).click();

  const dialog = page.getByRole("dialog", { name: "Add Task" });
  await dialog.getByLabel("Title").fill("Measure closet shelves");
  await dialog.getByLabel("Target").selectOption({ label: "Create unplaced room" });
  await dialog.getByLabel("Room Name").fill("Hall Closet");
  await dialog.getByRole("button", { name: "Create Task" }).click();

  await expect(page.getByRole("heading", { name: "Unplaced Rooms" })).toBeVisible();
  await expect(page.locator(".unplaced-panel").getByText("Hall Closet")).toBeVisible();
  await expect(taskInput(page, "Measure closet shelves")).toBeVisible();
});

test("progress control appears only after status moves in progress", async ({ page }) => {
  await openFresh(page);
  await createBlankHouse(page);
  await createTaskFromRoomPanel(page, "Repair loose railing");

  await expect(page.getByRole("spinbutton", { name: "Progress" })).toHaveCount(0);
  await page.getByRole("combobox", { name: "Status" }).selectOption("in-progress");
  await expect(page.getByRole("spinbutton", { name: "Progress" })).toBeVisible();
});

test("dirty draft restores after reload", async ({ page }) => {
  await openFresh(page);
  await createBlankHouse(page);
  await createTaskFromRoomPanel(page, "Order outlet covers");

  await expect(page.getByText("Unsaved changes")).toBeVisible();
  await page.reload();

  await expect(taskInput(page, "Order outlet covers")).toBeVisible();
  await expect(page.getByText("Unsaved changes")).toBeVisible();
});

test("start over clears the house and returns to setup flow", async ({ page }) => {
  await openFresh(page);
  await createBlankHouse(page);
  await createTaskFromRoomPanel(page, "Remove old shelf anchors");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByText("Cloud session")).toBeVisible();

  await page.getByRole("button", { name: "Start Over" }).click();

  await expect(page.getByRole("heading", { name: "Plan home projects by room, not just by list." })).toBeVisible();
  await expect(page.getByRole("button", { name: "Start with projects" })).toBeVisible();
  await page.reload();
  await expect(page.getByRole("heading", { name: "Plan home projects by room, not just by list." })).toBeVisible();
});
