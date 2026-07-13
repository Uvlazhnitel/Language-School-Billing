import { describe, expect, it } from "vitest";

import { defaultLessonPriceForCourseType } from "./courseDefaults";

describe("courseDefaults", () => {
  it("uses 15 EUR as the default lesson price for group courses", () => {
    expect(defaultLessonPriceForCourseType("group")).toBe("15.00");
  });

  it("uses 25 EUR as the default lesson price for individual courses", () => {
    expect(defaultLessonPriceForCourseType("individual")).toBe("25.00");
  });
});
