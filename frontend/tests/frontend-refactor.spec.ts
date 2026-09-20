import { test, expect } from "@playwright/test";
import { createTopic, login } from "./helpers/forum";

test("повтор ответа после потери сети не создаёт дубликат", async ({
  page,
}) => {
  await createTopic(page);
  const topicId = new URL(page.url()).pathname.split("/").pop();
  const text = "Ответ после потери сети " + Date.now();
  let lost = false;
  await page.route("**/api/v1/topics/*/posts", async (route) => {
    if (route.request().method() === "POST" && !lost) {
      lost = true;
      await route.fetch();
      await route.abort("failed");
    } else await route.continue();
  });
  await page.getByLabel("Ваш ответ").fill(text);
  await page.getByRole("button", { name: "Отправить ответ" }).click();
  await expect(page.getByRole("alert")).toContainText("Нет связи");
  await page.reload();
  await expect(page.getByLabel("Ваш ответ")).toHaveValue(text);
  await page.getByRole("button", { name: "Отправить ответ" }).click();
  await expect(page.getByLabel("Ваш ответ")).toHaveValue("");
  await expect(page.getByText(text, { exact: true })).toHaveCount(1);
  const response = await page.request.get(`/api/v1/topics/${topicId}/posts`);
  const data = await response.json();
  expect(
    data.items.filter((post: { body: string }) => post.body === text),
  ).toHaveLength(1);
});

test("мобильное меню закрывается при выборе раздела", async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto("/");
  await page.getByRole("button", { name: "Открыть меню" }).click();
  const section = page
    .getByRole("navigation", { name: "Разделы" })
    .getByRole("link")
    .filter({ hasText: "Технологии" });
  await section.click();
  await expect(page).toHaveURL(/section_id=/);
  await expect(page.locator(".sidebar")).not.toHaveClass(/is-open/);
  await expect(page.locator(".menu-scrim")).toHaveCount(0);
  await expect(
    page.getByRole("heading", { name: "Технологии.", exact: true }),
  ).toBeVisible();
});

test("черновики ответов изолированы при переходах между темами", async ({
  page,
}) => {
  await page.goto("/");
  const links = page.locator(".topic-row h2 a");
  await expect(links.first()).toBeVisible();
  const first = await links.nth(0).getAttribute("href");
  const second = await links.nth(1).getAttribute("href");
  await links.nth(0).click();
  await page.getByLabel("Ваш ответ").fill("Черновик первой темы");
  await page.goBack();
  await page.locator(`.topic-row h2 a[href="${second}"]`).click();
  await expect(page.getByLabel("Ваш ответ")).toHaveValue("");
  await page.getByLabel("Ваш ответ").fill("Черновик второй темы");
  await page.goBack();
  await page.locator(`.topic-row h2 a[href="${first}"]`).click();
  await expect(page.getByLabel("Ваш ответ")).toHaveValue(
    "Черновик первой темы",
  );
  await page.reload();
  await expect(page.getByLabel("Ваш ответ")).toHaveValue(
    "Черновик первой темы",
  );
});

test("модерация загружает только данные выбранной вкладки", async ({
  page,
}) => {
  const requests = { reports: 0, actions: 0 };
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (request.method() !== "GET") return;
    if (path === "/api/v1/mod/reports") requests.reports++;
    if (path === "/api/v1/mod/actions") requests.actions++;
  });
  const initialReports = page.waitForResponse((response) =>
    response.url().includes("/api/v1/mod/reports?"),
  );
  await login(page);
  await initialReports;
  expect(requests.reports).toBeGreaterThan(0);
  expect(requests.actions).toBe(0);
  const reportCount = requests.reports;
  const actionsLoaded = page.waitForResponse((response) =>
    response.url().includes("/api/v1/mod/actions?"),
  );
  await page.getByRole("button", { name: "Журнал действий" }).click();
  await actionsLoaded;
  expect(requests.actions).toBe(1);
  expect(requests.reports).toBe(reportCount);
  const actionCount = requests.actions;
  const resolvedLoaded = page.waitForResponse((response) =>
    response.url().includes("/api/v1/mod/reports?status=resolved"),
  );
  await page.getByRole("button", { name: "Скрытые", exact: true }).click();
  await resolvedLoaded;
  expect(requests.actions).toBe(actionCount);
  await expect(
    page.getByRole("button", { name: "Скрытые", exact: true }),
  ).toHaveClass("active");
});
