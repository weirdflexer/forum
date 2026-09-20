import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
const title = () => `Тестовая тема ${Date.now()}`;
async function createTopic(
  page: Page,
  name = title(),
  body = "Проверяем работу форума.",
) {
  await page.goto("/new");
  await page
    .getByLabel("Раздел", { exact: true })
    .selectOption({ label: "Технологии" });
  await page.getByLabel("Заголовок", { exact: true }).fill(name);
  await page.getByLabel("Первое сообщение", { exact: true }).fill(body);
  await page.getByRole("button", { name: "Опубликовать", exact: true }).click();
  await expect(page).toHaveURL(/\/topics\//);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(name);
  return name;
}
async function login(page: Page) {
  await page.goto("/login");
  await page
    .getByLabel("Логин", { exact: true })
    .fill(process.env.E2E_ADMIN_LOGIN!);
  await page
    .getByLabel("Пароль", { exact: true })
    .fill(process.env.E2E_ADMIN_PASSWORD!);
  await page.getByRole("button", { name: "Войти", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Модерация." })).toBeVisible();
}
test("публикация, ответ, удаление и очищенный черновик", async ({ page }) => {
  const name = await createTopic(page);
  await page.reload();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(name);
  await page.getByLabel("Ваш ответ").fill("Ответ сохраняется в PostgreSQL");
  await page.getByRole("button", { name: "Отправить ответ" }).click();
  await expect(
    page.getByText("Ответ сохраняется в PostgreSQL", { exact: true }),
  ).toBeVisible();
  await page
    .locator(".post")
    .last()
    .getByRole("button", { name: "Удалить", exact: true })
    .click();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Удалить сообщение", exact: true })
    .click();
  await expect(page.getByText("Сообщение удалено автором.")).toBeVisible();
  await page.goto("/new");
  await expect(page.getByLabel("Заголовок", { exact: true })).toHaveValue("");
});
test("потерянный ответ сервера и повтор без дубликата", async ({ page }) => {
  await page.goto("/new");
  const name = title();
  await page
    .getByLabel("Раздел", { exact: true })
    .selectOption({ label: "Учёба" });
  await page.getByLabel("Заголовок", { exact: true }).fill(name);
  await page
    .getByLabel("Первое сообщение", { exact: true })
    .fill("Сохранится даже при потере ответа сервера");
  let lost = false;
  await page.route("**/api/v1/topics", async (route) => {
    if (route.request().method() === "POST" && !lost) {
      lost = true;
      await route.fetch();
      await route.abort("failed");
    } else await route.continue();
  });
  await page.getByRole("button", { name: "Опубликовать", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("Нет связи");
  await page.reload();
  await expect(page.getByLabel("Заголовок", { exact: true })).toHaveValue(name);
  await page.getByRole("button", { name: "Опубликовать", exact: true }).click();
  await expect(page).toHaveURL(/\/topics\//);
  const resp = await page.request.get(
    "/api/v1/topics?q=" + encodeURIComponent(name),
  );
  expect((await resp.json()).total).toBe(1);
});
test("жалоба, решение модератора, скрытие и журнал", async ({
  page,
  browser,
}) => {
  const decisionReason = "Проверка жалобы " + Date.now();
  await createTopic(page, title(), "Секретнаяфразамодерации");
  const topicURL = page.url();
  await page.getByRole("button", { name: "Пожаловаться" }).click();
  await page
    .getByLabel("Причина", { exact: true })
    .selectOption("personal_data");
  await page.getByRole("button", { name: "Отправить жалобу" }).click();
  await expect(page.getByRole("status")).toContainText("Жалоба отправлена");
  const staffContext = await browser.newContext({
    baseURL: process.env.E2E_BASE_URL,
  });
  const staff = await staffContext.newPage();
  await login(staff);
  const report = staff
    .locator(".report-card")
    .filter({ hasText: "Секретнаяфразамодерации" });
  await report.getByRole("button", { name: "Скрыть сообщение" }).click();
  await staff
    .getByLabel("Причина решения — попадёт в журнал")
    .fill(decisionReason);
  await staff.getByRole("button", { name: "Сохранить решение" }).click();
  await expect(staff.locator(".toast")).toContainText("Решение сохранено");
  await page.goto(topicURL);
  await expect(page.getByText("Сообщение скрыто модератором.")).toBeVisible();
  await expect(
    page.getByText("Секретнаяфразамодерации", { exact: true }),
  ).toHaveCount(0);
  await staff.getByRole("button", { name: "Журнал действий" }).click();
  await expect(staff.getByText(decisionReason)).toBeVisible();
  await staffContext.close();
});
test("администратор создаёт раздел и архивирует его", async ({ page }) => {
  await login(page);
  await page.getByRole("link", { name: "Управление", exact: true }).click();
  await page.getByRole("button", { name: "Добавить раздел" }).click();
  const name = "Раздел " + Date.now();
  await page.getByLabel("Название", { exact: true }).fill(name);
  await page.getByLabel("Короткий адрес").fill("e2e-" + Date.now());
  await page
    .getByLabel("Описание", { exact: true })
    .fill("Проверка административного интерфейса");
  await page.getByRole("button", { name: "Сохранить", exact: true }).click();
  const row = page.locator(".admin-row").filter({ hasText: name });
  await expect(row).toBeVisible();
  await row.getByRole("button", { name: "Изменить", exact: true }).click();
  await page.getByLabel("В архиве — нельзя создавать новые темы").check();
  await page.getByRole("button", { name: "Сохранить", exact: true }).click();
  await expect(row.getByText("Архив", { exact: true })).toBeVisible();
  await page.goto("/new");
  await expect(
    page
      .getByLabel("Раздел", { exact: true })
      .locator("option")
      .filter({ hasText: name }),
  ).toHaveCount(0);
});
test("пользовательский HTML выводится текстом", async ({ page }) => {
  await createTopic(page, title(), "<script>window.hacked=true</script>");
  await expect(page.locator(".post-body").first()).toHaveText(
    "<script>window.hacked=true</script>",
  );
  expect(await page.evaluate(() => Object.hasOwn(window, "hacked"))).toBe(
    false,
  );
});
test("360, 768 и 1440 px: навигация без горизонтального переполнения", async ({
  page,
}) => {
  for (const width of [360, 768, 1440]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto("/");
    await expect(
      page.getByRole("heading", { name: "Все обсуждения." }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: `test-results/home-${width}.png`,
      fullPage: true,
    });
    if (width === 360) {
      await page.getByRole("button", { name: "Открыть меню" }).click();
      await expect(
        page.getByRole("navigation", { name: "Разделы" }),
      ).toBeVisible();
      await page
        .getByRole("button", { name: "Закрыть меню", exact: true })
        .first()
        .click();
    }
    await page.goto("/new");
    await expect(
      page.getByLabel("Первое сообщение", { exact: true }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
  }
  await page.setViewportSize({ width: 720, height: 600 });
  await page.goto("/");
  await page.keyboard.press("Tab");
  await expect(page.getByRole("link", { name: "К содержимому" })).toBeFocused();
});
