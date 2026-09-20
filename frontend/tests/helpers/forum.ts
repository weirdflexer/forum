import { expect } from "@playwright/test";
import type { Page } from "@playwright/test";

export const title = () => `Тестовая тема ${Date.now()}`;
export async function createTopic(
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
export async function login(page: Page) {
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
