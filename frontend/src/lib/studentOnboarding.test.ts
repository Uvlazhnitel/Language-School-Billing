import { describe, expect, it } from "vitest";

import type { CourseDTO } from "./courses";
import {
  buildOnboardingEnrollmentInputs,
  changeOnboardingBillingMode,
  createEmptyOnboardingEnrollmentRow,
  selectOnboardingCourse,
} from "./studentOnboarding";

const courses: CourseDTO[] = [
  {
    id: 1,
    version: 1,
    name: "Group Course",
    teacherName: "Teacher",
    type: "group",
    lessonPrice: 15,
    subscriptionPrice: 60,
  },
  {
    id: 2,
    version: 1,
    name: "Individual Course",
    teacherName: "Teacher",
    type: "individual",
    lessonPrice: 25,
    subscriptionPrice: 80,
  },
];

describe("student onboarding enrollment rows", () => {
  it("applies course defaults independently to each row", () => {
    const first = selectOnboardingCourse(createEmptyOnboardingEnrollmentRow(), 1, courses);
    const second = selectOnboardingCourse(createEmptyOnboardingEnrollmentRow(), 2, courses);

    expect(first).toMatchObject({ courseId: 1, lessonPrice: "15", chargeMaterials: true });
    expect(second).toMatchObject({ courseId: 2, lessonPrice: "25", chargeMaterials: true });
    expect(changeOnboardingBillingMode(second, "subscription", courses)).toMatchObject({
      billingMode: "subscription",
      subscriptionPrice: "80",
    });
  });

  it("ignores empty rows and preserves selected row order", () => {
    const first = selectOnboardingCourse(createEmptyOnboardingEnrollmentRow(), 2, courses);
    const empty = createEmptyOnboardingEnrollmentRow();
    const second = selectOnboardingCourse(createEmptyOnboardingEnrollmentRow(), 1, courses);

    expect(buildOnboardingEnrollmentInputs([first, empty, second]).inputs.map((item) => item.courseId)).toEqual([
      2, 1,
    ]);
  });

  it("rejects duplicate courses and invalid prices", () => {
    const first = selectOnboardingCourse(createEmptyOnboardingEnrollmentRow(), 1, courses);
    const duplicate = selectOnboardingCourse(createEmptyOnboardingEnrollmentRow(), 1, courses);
    expect(buildOnboardingEnrollmentInputs([first, duplicate]).errorKey).toBe(
      "msg.duplicateOnboardingCourse"
    );

    expect(
      buildOnboardingEnrollmentInputs([{ ...first, lessonPrice: "invalid" }]).errorKey
    ).toBe("msg.lessonPriceOverrideRange");
  });
});
