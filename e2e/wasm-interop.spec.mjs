import { test, expect } from "@playwright/test";

test("WASM API rejects malformed JavaScript values without crashing", async ({ page }) => {
  await page.goto("/?api=");
  await expect(page.locator("#wasmStatus")).toHaveText("Go WASM稼働中（距離計算）");

  const results = await page.evaluate(() => ({
    stringDistance: window.MinyoGo.distanceKm("34.7", 135.5, 34.8, 135.6),
    nullDistance: window.MinyoGo.distanceKm(null, 135.5, 34.8, 135.6),
    stringCoordinates: window.MinyoGo.validCoordinates("34.7", 135.5),
    nonArrayAnswers: window.MinyoGo.validateAnswerIDs("1,2,3", [1, 2, 3]),
    stringAnswerID: window.MinyoGo.validateAnswerIDs([1, 2, 3], [1, "2", 3]),
    fractionalAnswerID: window.MinyoGo.validateAnswerIDs([1, 2, 3], [1, 2.5, 3]),
    stringPassedCount: window.MinyoGo.passed("3", 3),
    fractionalPassedCount: window.MinyoGo.passed(3.5, 3),
  }));

  expect(results).toEqual({
    stringDistance: null,
    nullDistance: null,
    stringCoordinates: false,
    nonArrayAnswers: false,
    stringAnswerID: false,
    fractionalAnswerID: false,
    stringPassedCount: false,
    fractionalPassedCount: false,
  });

  expect(await page.evaluate(() => window.MinyoGo.version)).toBe("0.1.0");
});
