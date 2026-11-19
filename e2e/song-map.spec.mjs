import { test, expect } from "@playwright/test";

test("Go WASM loads in the real browser and exposes distance calculation", async ({ page }) => {
  const wasmResponse = page.waitForResponse(response =>
    response.url().endsWith("/wasm/minyo-go.wasm")
  );
  await page.goto("/?api=");
  const response = await wasmResponse;
  expect(response.ok()).toBeTruthy();
  expect(response.headers()["content-type"]).toContain("application/wasm");
  await expect(page.locator("#wasmStatus")).toHaveText("Go WASM稼働中（距離計算）");
  const distance = await page.evaluate(() =>
    window.MinyoGo.distanceKm(34.6937, 135.5023, 34.6525, 135.5063)
  );
  expect(distance).toBeGreaterThan(0);
  expect(distance).toBeLessThan(10);
  const apiChecks = await page.evaluate(() => ({
    validCoordinates: window.MinyoGo.validCoordinates(34.7, 135.5),
    invalidCoordinates: window.MinyoGo.validCoordinates(91, 135.5),
    validAnswerIDs: window.MinyoGo.validateAnswerIDs([1, 2, 3], [3, 2, 1]),
    duplicateAnswerIDs: window.MinyoGo.validateAnswerIDs([1, 2, 3], [1, 1, 3]),
    passed: window.MinyoGo.passed(3, 3),
    failed: window.MinyoGo.passed(2, 3),
  }));
  expect(apiChecks).toEqual({
    validCoordinates: true,
    invalidCoordinates: false,
    validAnswerIDs: true,
    duplicateAnswerIDs: false,
    passed: true,
    failed: false,
  });
});

test("JavaScript fallback remains available when the WASM download fails", async ({ page }) => {
  await page.route("**/wasm/minyo-go.wasm", route => route.abort());
  await page.goto("/?api=");
  await expect(page.locator("#wasmStatus")).toHaveText("Go WASM未使用（JS計算で継続）");
  await expect(page.locator("#list .card").first()).toBeVisible();
  expect(await page.evaluate(() => typeof window.MinyoGo)).toBe("undefined");
});

test("song map opens three questions and round-trips an answer attempt to the API", async ({ page }) => {
  await page.goto("/?api=");
  await expect(page.locator("#apiStatus")).toContainText("API接続中");
  await expect(page.locator("[data-quiz]").first()).toBeVisible();
  const spotID = await page.locator("[data-quiz]").first().getAttribute("data-quiz");
  expect(spotID).toBeTruthy();
  const quizResponse = await page.request.get(`/api/spots/${encodeURIComponent(spotID)}/quiz`);
  expect(quizResponse.ok()).toBeTruthy();
  expect(await quizResponse.text()).not.toContain("answer_index");
  await page.locator("[data-quiz]").first().click();
  await expect(page.locator("#quizDialog")).toBeVisible();
  await expect(page.locator("#quizQuestions .quiz-question")).toHaveCount(3);

  await page.locator("#testToggle").check();
  for (const question of await page.locator("#quizQuestions .quiz-question").all()) {
    await question.locator('input[type="radio"]').first().check();
  }
  await page.locator("#quizSubmit").click();
  await expect(page.locator("#quizStatus")).toContainText(/\d\/3問正解|スタンプを獲得|取得済み/);

  const userID = await page.evaluate(() => localStorage.getItem("minyo-go.apiUserId.v1"));
  expect(userID).toBeTruthy();
  const stampsResponse = await page.request.get(`/api/users/${encodeURIComponent(userID)}/stamps`);
  expect(stampsResponse.ok()).toBeTruthy();
  const stamps = await stampsResponse.json();
  expect(stamps).toHaveProperty("spots");
});
