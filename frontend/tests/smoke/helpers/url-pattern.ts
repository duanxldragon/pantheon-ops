import { expect, type Page } from '@playwright/test';

export function expectPagePathname(page: Page, pathname: string) {
  expect(new URL(page.url()).pathname).toBe(pathname);
}
