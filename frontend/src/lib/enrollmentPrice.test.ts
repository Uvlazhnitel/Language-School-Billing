import { describe, expect, it } from "vitest";

import type { EnrollmentDTO } from "./enrollments";
import {
  enrollmentPrice,
  enrollmentPriceUpdateValues,
  parseEnrollmentPrice,
} from "./enrollmentPrice";

const enrollment: EnrollmentDTO = {
  id: 1,
  version: 1,
  studentId: 1,
  studentName: "Student",
  courseId: 1,
  courseName: "Course",
  courseType: "individual",
  teacherName: "Teacher",
  billingMode: "per_lesson",
  chargeMaterials: true,
  lessonPriceOverride: 25,
  subscriptionLessonPrice: 90,
  note: "Keep this note",
  createdAt: "2026-07-01T10:00:00Z",
};

describe("enrollmentPrice", () => {
  it("reads the editable price for both billing modes", () => {
    expect(enrollmentPrice(enrollment)).toBe(25);
    expect(enrollmentPrice({ ...enrollment, billingMode: "subscription" })).toBe(90);
  });
});

describe("parseEnrollmentPrice", () => {
  it("accepts non-negative amounts with up to two decimals", () => {
    expect(parseEnrollmentPrice("25.50")).toBe(25.5);
    expect(parseEnrollmentPrice("15,25")).toBe(15.25);
    expect(parseEnrollmentPrice("0")).toBe(0);
    expect(parseEnrollmentPrice(" ")).toBe(0);
  });

  it("rejects negative, non-numeric, and over-precise amounts", () => {
    expect(parseEnrollmentPrice("-1")).toBeNull();
    expect(parseEnrollmentPrice("abc")).toBeNull();
    expect(parseEnrollmentPrice("12.345")).toBeNull();
  });
});

describe("enrollmentPriceUpdateValues", () => {
  it("changes only the per-lesson price and preserves the other enrollment terms", () => {
    expect(enrollmentPriceUpdateValues(enrollment, 30)).toEqual({
      billingMode: "per_lesson",
      chargeMaterials: true,
      lessonPriceOverride: 30,
      subscriptionLessonPrice: 0,
      note: "Keep this note",
    });
  });

  it("changes only the subscription price and preserves the other enrollment terms", () => {
    expect(
      enrollmentPriceUpdateValues({ ...enrollment, billingMode: "subscription" }, 110)
    ).toEqual({
      billingMode: "subscription",
      chargeMaterials: true,
      lessonPriceOverride: 0,
      subscriptionLessonPrice: 110,
      note: "Keep this note",
    });
  });
});
