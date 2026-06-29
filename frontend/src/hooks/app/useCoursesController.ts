import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { createCourse, deleteCourse, listCourses, updateCourse, type CourseDTO } from "../../lib/courses";
import { createTeacher, listTeachers, type TeacherDTO } from "../../lib/teachers";
import { decimalOrZero, normalizeMoneyInput } from "../../lib/appUi";
import { type TranslateFn } from "../../lib/i18n";
import { isConflictError } from "../../lib/api/shared";

function isStaleRevisionError(error: unknown): boolean {
  const message = String((error as { message?: string } | undefined)?.message ?? error ?? "").toLowerCase();
  return (isConflictError(error) || message.includes("conflict")) && message.includes("record was changed or deleted by another user");
}

type UseCoursesControllerParams = {
  appReady: boolean;
  tab: string;
  showMessage: (message: string, type?: "success" | "error") => void;
  showConfirm: (message: string, onConfirm: () => void | Promise<void>) => void;
  t: TranslateFn;
};

export function useCoursesController({
  appReady,
  tab,
  showMessage,
  showConfirm,
  t,
}: UseCoursesControllerParams) {
  const [allCourses, setAllCourses] = useState<CourseDTO[]>([]);
  const [courseQ, setCourseQ] = useState("");
  const [courseTypeFilter, setCourseTypeFilter] = useState<"" | "group" | "individual">("");
  const [courseTeacherFilter, setCourseTeacherFilter] = useState("all");
  const [coursePricingFilter, setCoursePricingFilter] = useState<
    "all" | "lesson" | "subscription" | "both" | "lesson_only" | "subscription_only"
  >("all");
  const [courseLoading, setCourseLoading] = useState(false);
  const [courseModalOpen, setCourseModalOpen] = useState(false);
  const [editingCourse, setEditingCourse] = useState<CourseDTO | null>(null);
  const [cfName, setCfName] = useState("");
  const [cfTeacherId, setCfTeacherId] = useState<number | undefined>(undefined);
  const [cfTeacherSearch, setCfTeacherSearch] = useState("");
  const [cfTeacherPickerOpen, setCfTeacherPickerOpen] = useState(false);
  const [cfTeacherCreating, setCfTeacherCreating] = useState(false);
  const [allTeachers, setAllTeachers] = useState<TeacherDTO[]>([]);
  const cfTeacherComboRef = useRef<HTMLDivElement | null>(null);
  const [cfType, setCfType] = useState<"group" | "individual">("group");
  const [cfLessonPrice, setCfLessonPrice] = useState("0.00");
  const [cfSubscriptionPrice, setCfSubscriptionPrice] = useState("0.00");

  const handleCoursePriceChange = useCallback((value: string, setter: (value: string) => void) => {
    const next = normalizeMoneyInput(value);
    if (next !== null) setter(next);
  }, []);

  const loadAllCourses = useCallback(async () => {
    const data = await listCourses("");
    setAllCourses(data);
    return data;
  }, []);

  const loadCourses = useCallback(async () => {
    setCourseLoading(true);
    try {
      return await loadAllCourses();
    } finally {
      setCourseLoading(false);
    }
  }, [loadAllCourses]);

  const loadAllTeachers = useCallback(async () => {
    const data = await listTeachers("");
    setAllTeachers(data);
    return data;
  }, []);

  useEffect(() => {
    if (!appReady) return;
    if (tab === "courses") void loadCourses();
  }, [appReady, tab, loadCourses]);

  useEffect(() => {
    if (!appReady) return;
    void loadAllCourses();
  }, [appReady, loadAllCourses]);

  useEffect(() => {
    if (!appReady) return;
    void loadAllTeachers();
  }, [appReady, loadAllTeachers]);

  const selectedCourseTeacher = useMemo(
    () => allTeachers.find((teacher) => teacher.id === cfTeacherId) ?? null,
    [allTeachers, cfTeacherId],
  );

  const filteredTeachers = useMemo(() => {
    const q = cfTeacherSearch.trim().toLowerCase();
    if (!q) return allTeachers;
    return allTeachers.filter((teacher) => teacher.fullName.toLowerCase().includes(q));
  }, [allTeachers, cfTeacherSearch]);

  const exactTeacherMatch = useMemo(() => {
    const q = cfTeacherSearch.trim().toLowerCase();
    if (!q) return null;
    return allTeachers.find((teacher) => teacher.fullName.trim().toLowerCase() === q) ?? null;
  }, [allTeachers, cfTeacherSearch]);

  const courseTeacherOptions = useMemo(() => {
    const seen = new Map<number, string>();
    for (const course of allCourses) {
      if (!course.teacherId || !course.teacherName) continue;
      if (!seen.has(course.teacherId)) {
        seen.set(course.teacherId, course.teacherName);
      }
    }
    return Array.from(seen.entries())
      .map(([id, label]) => ({ value: String(id), label }))
      .sort((left, right) => left.label.localeCompare(right.label));
  }, [allCourses]);

  const courseList = useMemo(() => {
    const q = courseQ.trim().toLowerCase();
    return allCourses.filter((course) => {
      if (q) {
        const haystack = `${course.name} ${course.teacherName || ""}`.toLowerCase();
        if (!haystack.includes(q)) return false;
      }
      if (courseTypeFilter && course.type !== courseTypeFilter) return false;
      if (courseTeacherFilter === "none") {
        if (course.teacherId || course.teacherName?.trim()) return false;
      } else if (courseTeacherFilter !== "all" && String(course.teacherId ?? "") !== courseTeacherFilter) {
        return false;
      }

      const hasLessonPrice = course.lessonPrice > 0;
      const hasSubscriptionPrice = course.subscriptionPrice > 0;
      switch (coursePricingFilter) {
        case "lesson":
          if (!hasLessonPrice) return false;
          break;
        case "subscription":
          if (!hasSubscriptionPrice) return false;
          break;
        case "both":
          if (!hasLessonPrice || !hasSubscriptionPrice) return false;
          break;
        case "lesson_only":
          if (!hasLessonPrice || hasSubscriptionPrice) return false;
          break;
        case "subscription_only":
          if (!hasSubscriptionPrice || hasLessonPrice) return false;
          break;
      }

      return true;
    });
  }, [allCourses, coursePricingFilter, courseQ, courseTeacherFilter, courseTypeFilter]);

  const courseFiltersActive = Boolean(
    courseQ.trim() || courseTypeFilter || courseTeacherFilter !== "all" || coursePricingFilter !== "all",
  );

  const clearCourseFilters = useCallback(() => {
    setCourseQ("");
    setCourseTypeFilter("");
    setCourseTeacherFilter("all");
    setCoursePricingFilter("all");
  }, []);

  useEffect(() => {
    if (!cfTeacherPickerOpen) return;

    const handlePointerDown = (event: MouseEvent) => {
      if (!cfTeacherComboRef.current?.contains(event.target as Node)) {
        setCfTeacherPickerOpen(false);
      }
    };

    document.addEventListener("mousedown", handlePointerDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
    };
  }, [cfTeacherPickerOpen]);

  const openAddCourse = useCallback(() => {
    setEditingCourse(null);
    setCfName("");
    setCfTeacherId(undefined);
    setCfTeacherSearch("");
    setCfTeacherPickerOpen(false);
    setCfType("group");
    setCfLessonPrice("");
    setCfSubscriptionPrice("");
    setCourseModalOpen(true);
  }, []);

  const openEditCourse = useCallback((course: CourseDTO) => {
    setEditingCourse(course);
    setCfName(course.name);
    setCfTeacherId(course.teacherId);
    setCfTeacherSearch(course.teacherName);
    setCfTeacherPickerOpen(false);
    setCfType(course.type);
    setCfLessonPrice(course.lessonPrice.toFixed(2));
    setCfSubscriptionPrice(course.subscriptionPrice.toFixed(2));
    setCourseModalOpen(true);
  }, []);

  const addTeacherFromCourseForm = useCallback(async () => {
    const name = cfTeacherSearch.trim();
    if (!name) return;

    try {
      setCfTeacherCreating(true);
      const created = await createTeacher(name);
      setAllTeachers((prev) => {
        const withoutSame = prev.filter((teacher) => teacher.id !== created.id);
        return [...withoutSame, created].sort((left, right) => left.fullName.localeCompare(right.fullName));
      });
      setCfTeacherId(created.id);
      setCfTeacherSearch(created.fullName);
      setCfTeacherPickerOpen(false);
      showMessage(t("msg.teacherAdded"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    } finally {
      setCfTeacherCreating(false);
    }
  }, [cfTeacherSearch, showMessage, t]);

  const saveCourse = useCallback(async () => {
    const lessonPrice = decimalOrZero(cfLessonPrice);
    const subscriptionPrice = decimalOrZero(cfSubscriptionPrice);
    const trimmedTeacherSearch = cfTeacherSearch.trim();

    if (!cfName.trim()) {
      showMessage(t("msg.courseNameRequired"), "error");
      return;
    }
    if (lessonPrice < 0 || subscriptionPrice < 0) {
      showMessage(t("msg.coursePricesNonNegative"), "error");
      return;
    }

    let teacherId = cfTeacherId;
    if (!teacherId && exactTeacherMatch) {
      teacherId = exactTeacherMatch.id;
    }
    if (trimmedTeacherSearch && !teacherId) {
      showMessage(t("msg.courseTeacherRequired"), "error");
      return;
    }

    try {
      if (editingCourse) {
        await updateCourse(
          editingCourse.id,
          editingCourse.version,
          cfName,
          teacherId,
          cfType,
          lessonPrice,
          subscriptionPrice,
        );
      } else {
        await createCourse(cfName, teacherId, cfType, lessonPrice, subscriptionPrice);
      }

      setCourseModalOpen(false);
      await loadCourses();
      showMessage(editingCourse ? t("msg.courseUpdated") : t("msg.courseCreated"));
    } catch (e: any) {
      if (isStaleRevisionError(e)) {
        showMessage(t("msg.recordConflict"), "error");
        return;
      }
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }, [
    cfLessonPrice,
    cfName,
    cfSubscriptionPrice,
    cfTeacherId,
    cfTeacherSearch,
    cfType,
    editingCourse,
    exactTeacherMatch,
    loadCourses,
    showMessage,
    t,
  ]);

  const removeCourse = useCallback(
    async (id: number) => {
      showConfirm(t("msg.courseDeleteConfirm"), async () => {
        try {
          const currentCourse =
            courseList.find((item) => item.id === id) ?? allCourses.find((item) => item.id === id);
          if (!currentCourse) {
            throw new Error(t("msg.courseNotFound"));
          }
          await deleteCourse(id, currentCourse.version);
          await loadCourses();
          showMessage(t("msg.courseDeleted"));
        } catch (e: any) {
          if (isStaleRevisionError(e)) {
            showMessage(t("msg.recordConflict"), "error");
            return;
          }
          showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
        }
      });
    },
    [allCourses, courseList, loadCourses, showConfirm, showMessage, t],
  );

  return {
    allCourses,
    allTeachers,
    courseList,
    courseQ,
    courseTypeFilter,
    courseTeacherFilter,
    coursePricingFilter,
    courseLoading,
    courseModalOpen,
    editingCourse,
    cfName,
    cfTeacherId,
    cfTeacherSearch,
    cfTeacherPickerOpen,
    cfTeacherCreating,
    cfTeacherComboRef,
    cfType,
    cfLessonPrice,
    cfSubscriptionPrice,
    selectedCourseTeacher,
    filteredTeachers,
    exactTeacherMatch,
    courseTeacherOptions,
    courseFiltersActive,
    setCourseQ,
    setCourseTypeFilter,
    setCourseTeacherFilter,
    setCoursePricingFilter,
    setCfName,
    setCfTeacherSearch,
    setCfTeacherId,
    setCfTeacherPickerOpen,
    setCfType,
    setCourseModalOpen,
    clearCourseFilters,
    handleCoursePriceChange,
    openAddCourse,
    openEditCourse,
    addTeacherFromCourseForm,
    saveCourse,
    removeCourse,
    loadAllCourses,
    loadCourses,
  };
}
