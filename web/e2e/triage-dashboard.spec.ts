/**
 * DBIA Triage Dashboard — Playwright E2E spec
 *
 * Prerequisites:
 *   Go server running on :8080 with triage test data injected
 *   Frontend built and served from web/dist
 *
 * Run:
 *   npx playwright test web/e2e/triage-dashboard.spec.ts
 */

import { test, expect } from '@playwright/test';

const BASE = 'http://127.0.0.1:8080';

test.describe('Triage Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to triage page
    await page.goto(`${BASE}/triage`);
    // Wait for the page to load
    await page.waitForLoadState('networkidle');
  });

  test('loads without console errors', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(msg.text());
    });
    page.on('pageerror', err => errors.push(err.message));

    await page.goto(`${BASE}/triage`);
    await page.waitForLoadState('networkidle');

    // Wait a bit for any deferred errors
    await page.waitForTimeout(2000);

    expect(errors).toEqual([]);
  });

  test('displays triage dashboard heading', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Triage Dashboard' })).toBeVisible();
  });

  test('displays task cards or empty state', async ({ page }) => {
    // Either task cards or the "no tasks" empty state should be visible
    const taskCards = page.getByTestId('triage-task-card');
    const emptyState = page.getByTestId('triage-empty-state');

    // Wait for either condition
    await expect(taskCards.first().or(emptyState)).toBeVisible({ timeout: 10000 });
  });

  test('select all checkbox works', async ({ page }) => {
    // Only test if there are tasks
    const emptyState = page.getByTestId('triage-empty-state');
    const isEmpty = await emptyState.isVisible().catch(() => false);
    if (isEmpty) {
      test.skip();
      return;
    }

    const selectAll = page.getByTestId('triage-select-all');
    await selectAll.click();

    // Batch action bar should appear
    await expect(page.locator('text=批量 Approve')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=批量 Retry')).toBeVisible();
    await expect(page.locator('text=批量 Won\'t Fix')).toBeVisible();
  });

  test('individual checkbox toggles selection', async ({ page }) => {
    const emptyState = page.getByTestId('triage-empty-state');
    const isEmpty = await emptyState.isVisible().catch(() => false);
    if (isEmpty) {
      test.skip();
      return;
    }

    // Click the first task checkbox (not the select-all)
    const checkboxes = page.getByTestId('triage-task-checkbox');
    const count = await checkboxes.count();
    if (count < 1) {
      test.skip();
      return;
    }

    await checkboxes.first().click();

    // Batch bar should show "已选 1 项"
    await expect(page.locator('text=已选 1 项')).toBeVisible({ timeout: 5000 });
  });

  test('group mode renders grouped task sections', async ({ page }) => {
    const emptyState = page.getByTestId('triage-empty-state');
    const isEmpty = await emptyState.isVisible().catch(() => false);
    if (isEmpty) {
      test.skip();
      return;
    }

    await page.getByTestId('triage-group-mode').selectOption('status');
    await expect(page.getByTestId('triage-task-group-header').first()).toBeVisible({ timeout: 5000 });

    await page.getByTestId('triage-group-mode').selectOption('failure_code');
    await expect(page.getByTestId('triage-task-group-header').first()).toBeVisible({ timeout: 5000 });
  });

  test('wontfix actions require confirmation', async ({ page }) => {
    const emptyState = page.getByTestId('triage-empty-state');
    const isEmpty = await emptyState.isVisible().catch(() => false);
    if (isEmpty) {
      test.skip();
      return;
    }

    await page.getByTestId('triage-task-expand').first().click();
    await page.getByRole('button', { name: "Won't Fix" }).first().click();
    await expect(page.getByTestId('wontfix-confirm-dialog')).toBeVisible({ timeout: 5000 });
    await page.getByTestId('wontfix-cancel').click();
    await expect(page.getByTestId('wontfix-confirm-dialog')).toBeHidden();

    await page.getByTestId('triage-task-checkbox').first().click();
    await page.getByRole('button', { name: /批量 Won't Fix/ }).click();
    await expect(page.getByTestId('wontfix-confirm-dialog')).toBeVisible({ timeout: 5000 });
    await page.getByTestId('wontfix-cancel').click();
    await expect(page.getByTestId('wontfix-confirm-dialog')).toBeHidden();
  });

  test('expand task card shows details', async ({ page }) => {
    const emptyState = page.getByTestId('triage-empty-state');
    const isEmpty = await emptyState.isVisible().catch(() => false);
    if (isEmpty) {
      test.skip();
      return;
    }

    // Click on the first task card header to expand
    await page.getByTestId('triage-task-expand').first().click();

    // Should show detail fields
    await expect(page.locator('text=Review Decision').first()).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=Approve Manually').first()).toBeVisible();
    await expect(page.locator('text=Retry').first()).toBeVisible();
    await expect(page.locator('text=Won\'t Fix').first()).toBeVisible();
    await expect(page.locator('text=Timeline').first()).toBeVisible();
  });

  test('no 4xx/5xx network errors on load', async ({ page }) => {
    const failedRequests: string[] = [];
    page.on('response', response => {
      if (response.status() >= 400) {
        failedRequests.push(`${response.status()} ${response.url()}`);
      }
    });

    await page.goto(`${BASE}/triage`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // Filter out favicon (expected 404 on some setups)
    const realFailures = failedRequests.filter(r => !r.includes('favicon'));
    expect(realFailures).toEqual([]);
  });
});
