import { useEffect, useMemo, useState, useCallback, useRef } from "react";
import "./App.css";

import { fetchRows, saveCount, addOneMass, deleteEnrollment, Row } from "./lib/attendance";

import {
  listInvoices,
  getInvoice,
  genDrafts,
  issueOne,
  issueAll,
  ensurePdfAndOpen,
  InvoiceListItem,
  InvoiceDTO,
} from "./lib/invoices";

import {
  listStudents,
  createStudent,
  updateStudent,
  setStudentActive,
  deleteStudent,
  StudentDTO,
} from "./lib/students";

import { listCourses, createCourse, updateCourse, deleteCourse, CourseDTO } from "./lib/courses";

import { listEnrollments, createEnrollment, updateEnrollment, EnrollmentDTO } from "./lib/enrollments";

import { listDebtors, DebtorDTO, createPayment, invoiceSummary, InvoiceSummaryDTO } from "./lib/payments";

const months = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

type Tab = "students" | "courses" | "enrollments" | "attendance" | "invoice" | "debtors";

function numOrZero(s: string): number {
  if (s.trim() === "") return 0;
  const n = Number(s);
  return Number.isFinite(n) ? n : 0;
}

function intOrUndef(s: string): number | undefined {
  if (s.trim() === "") return undefined;
  const n = Number(s);
  return Number.isFinite(n) ? Math.trunc(n) : undefined;
}

export default function App() {
  const now = new Date();
  const [tab, setTab] = useState<Tab>("students");

  // Global message display
  const [message, setMessage] = useState<{ text: string; type: "success" | "error" } | null>(null);
  const messageTimeoutRef = useRef<number | null>(null);

  // Global confirmation dialog
  const [confirmDialog, setConfirmDialog] = useState<{
    isOpen: boolean;
    message: string;
    onConfirm: () => void | Promise<void>;
    confirmButtonLabel?: string;
  } | null>(null);

  const showConfirm = (messageText: string, onConfirm: () => void | Promise<void>, confirmButtonLabel?: string) => {
    setConfirmDialog({ isOpen: true, message: messageText, onConfirm, confirmButtonLabel });
  };

  const handleConfirmYes = async () => {
    try {
      if (confirmDialog?.onConfirm) {
        await confirmDialog.onConfirm();
      }
    } finally {
      setConfirmDialog(null);
    }
  };

  const handleConfirmNo = () => {
    setConfirmDialog(null);
  };

  const showMessage = useCallback((text: string, type: "success" | "error" = "success") => {
    console.log(`[${type.toUpperCase()}] ${text}`);

    // Clear any existing timeout
    if (messageTimeoutRef.current) {
      clearTimeout(messageTimeoutRef.current);
      messageTimeoutRef.current = null;
    }

    setMessage({ text, type });

    // Auto-dismiss success messages after 5 seconds
    if (type === "success") {
      messageTimeoutRef.current = window.setTimeout(() => {
        setMessage(null);
        messageTimeoutRef.current = null;
      }, 5000);
    }
  }, []);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (messageTimeoutRef.current) {
        clearTimeout(messageTimeoutRef.current);
      }
    };
  }, []);

  // Shared month/year for Attendance + Invoices
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth() + 1);

  // ---------------- Students ----------------
  const [students, setStudents] = useState<StudentDTO[]>([]);
  const [studentQ, setStudentQ] = useState("");
  const [includeInactive, setIncludeInactive] = useState(false);
  const [studentLoading, setStudentLoading] = useState(false);

  const [studentModalOpen, setStudentModalOpen] = useState(false);
  const [editingStudent, setEditingStudent] = useState<StudentDTO | null>(null);
  const [sfName, setSfName] = useState("");
  const [sfPhone, setSfPhone] = useState("");
  const [sfEmail, setSfEmail] = useState("");
  const [sfNote, setSfNote] = useState("");

  const loadStudents = useCallback(async () => {
    setStudentLoading(true);
    try {
      const data = await listStudents(studentQ, includeInactive);
      setStudents(data);
    } finally {
      setStudentLoading(false);
    }
  }, [studentQ, includeInactive]);

  useEffect(() => {
    if (tab === "students") loadStudents();
  }, [tab, loadStudents]);

  function openAddStudent() {
    setEditingStudent(null);
    setSfName("");
    setSfPhone("");
    setSfEmail("");
    setSfNote("");
    setStudentModalOpen(true);
  }

  function openEditStudent(s: StudentDTO) {
    setEditingStudent(s);
    setSfName(s.fullName);
    setSfPhone(s.phone);
    setSfEmail(s.email);
    setSfNote(s.note);
    setStudentModalOpen(true);
  }

  async function saveStudent() {
    if (!sfName.trim()) {
      showMessage("Full name is required", "error");
      return;
    }
    try {
      if (editingStudent) {
        // Update existing student
        await updateStudent(editingStudent.id, sfName, sfPhone, sfEmail, sfNote);
      } else {
        // Create new student
        await createStudent(sfName, sfPhone, sfEmail, sfNote);
      }
      setStudentModalOpen(false);
      await loadStudents(); // Refresh list
      showMessage(editingStudent ? "Student updated successfully!" : "Student created successfully!");
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  }

  async function toggleStudentActive(s: StudentDTO) {
    try {
      await setStudentActive(s.id, !s.isActive);
      await loadStudents();
      showMessage(s.isActive ? "Student deactivated" : "Student activated");
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  }

  async function removeStudent(id: number) {
    showConfirm(
      "Delete student? This will automatically remove their enrollments and attendance records. Deletion will fail if the student has any invoices or payments. This action cannot be undone.",
      async () => {
        try {
          await deleteStudent(id);
          await loadStudents();
          showMessage("Student deleted successfully!");
        } catch (e: any) {
          showMessage(`Error: ${String(e?.message ?? e)}`, "error");
        }
      }
    );
  }

  // ---------------- Courses ----------------
  const [courses, setCourses] = useState<CourseDTO[]>([]);
  const [courseQ, setCourseQ] = useState("");
  const [courseLoading, setCourseLoading] = useState(false);

  const [courseModalOpen, setCourseModalOpen] = useState(false);
  const [editingCourse, setEditingCourse] = useState<CourseDTO | null>(null);
  const [cfName, setCfName] = useState("");
  const [cfType, setCfType] = useState<"group" | "individual">("group");
  const [cfLessonPrice, setCfLessonPrice] = useState(0);
  const [cfSubscriptionPrice, setCfSubscriptionPrice] = useState(0);

  const loadCourses = useCallback(async () => {
    setCourseLoading(true);
    try {
      const data = await listCourses(courseQ);
      setCourses(data);
    } finally {
      setCourseLoading(false);
    }
  }, [courseQ]);

  useEffect(() => {
    if (tab === "courses") loadCourses();
  }, [tab, loadCourses]);

  function openAddCourse() {
    setEditingCourse(null);
    setCfName("");
    setCfType("group");
    setCfLessonPrice(0);
    setCfSubscriptionPrice(0);
    setCourseModalOpen(true);
  }

  function openEditCourse(c: CourseDTO) {
    setEditingCourse(c);
    setCfName(c.name);
    setCfType(c.type);
    setCfLessonPrice(c.lessonPrice);
    setCfSubscriptionPrice(c.subscriptionPrice);
    setCourseModalOpen(true);
  }

  async function saveCourse() {
    if (!cfName.trim()) {
      showMessage("Course name is required", "error");
      return;
    }
    if (cfLessonPrice < 0 || cfSubscriptionPrice < 0) {
      showMessage("Prices must be >= 0", "error");
      return;
    }

    try {
      if (editingCourse) {
        await updateCourse(editingCourse.id, cfName, cfType, cfLessonPrice, cfSubscriptionPrice);
      } else {
        await createCourse(cfName, cfType, cfLessonPrice, cfSubscriptionPrice);
      }

      setCourseModalOpen(false);
      await loadCourses();
      showMessage(editingCourse ? "Course updated successfully!" : "Course created successfully!");
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  }

  async function removeCourse(id: number) {
    showConfirm("Delete course? This is blocked if enrollments exist.", async () => {
      try {
        await deleteCourse(id);
        await loadCourses();
        showMessage("Course deleted successfully!");
      } catch (e: any) {
        showMessage(`Error: ${String(e?.message ?? e)}`, "error");
      }
    });
  }

  // ---------------- Enrollments ----------------
  const [enrollments, setEnrollments] = useState<EnrollmentDTO[]>([]);
  const [enrStudentFilter, setEnrStudentFilter] = useState<number | undefined>(undefined);
  const [enrCourseFilter, setEnrCourseFilter] = useState<number | undefined>(undefined);
  const [enrLoading, setEnrLoading] = useState(false);

  const [enrModalOpen, setEnrModalOpen] = useState(false);
  const [editingEnr, setEditingEnr] = useState<EnrollmentDTO | null>(null);
  const [efStudentId, setEfStudentId] = useState<number>(0);
  const [efCourseId, setEfCourseId] = useState<number>(0);
  const [efMode, setEfMode] = useState<"subscription" | "per_lesson">("per_lesson");
  const [efDiscount, setEfDiscount] = useState(0);
  const [efNote, setEfNote] = useState("");

  const loadEnrollments = useCallback(async () => {
    setEnrLoading(true);
    try {
      await Promise.all([
        students.length === 0 ? listStudents("", true).then(setStudents) : Promise.resolve(),
        courses.length === 0 ? listCourses("").then(setCourses) : Promise.resolve(),
      ]);

      const data = await listEnrollments(enrStudentFilter, enrCourseFilter);
      setEnrollments(data);
    } finally {
      setEnrLoading(false);
    }
  }, [enrStudentFilter, enrCourseFilter, students.length, courses.length]);

  useEffect(() => {
    if (tab === "enrollments") loadEnrollments();
  }, [tab, loadEnrollments]);

  function openAddEnrollment() {
    const activeStudents = students.filter((s) => s.isActive);
    if (activeStudents.length === 0) {
      showMessage("No active students available. Please add or activate students first.", "error");
      setTab("students");
      return;
    }
    if (courses.length === 0) {
      showMessage("No courses available. Please add courses first.", "error");
      setTab("courses");
      return;
    }

    const initialStudentId = activeStudents[0]?.id ?? 0;
    const initialCourseId = courses[0]?.id ?? 0;

    setEditingEnr(null);
    setEfStudentId(initialStudentId);
    setEfCourseId(initialCourseId);
    setEfMode("per_lesson");
    setEfDiscount(0);
    setEfNote("");
    setEnrModalOpen(true);
  }

  function openEditEnrollment(e: EnrollmentDTO) {
    setEditingEnr(e);
    setEfStudentId(e.studentId);
    setEfCourseId(e.courseId);
    setEfMode(e.billingMode);
    setEfDiscount(e.discountPct);
    setEfNote(e.note);
    setEnrModalOpen(true);
  }

  async function saveEnrollment() {
    if (efStudentId <= 0 || efCourseId <= 0) {
      showMessage("Please select both student and course", "error");
      return;
    }
    if (efDiscount < 0 || efDiscount > 100) {
      showMessage("Discount must be between 0 and 100", "error");
      return;
    }

    try {
      let result: EnrollmentDTO;
      if (editingEnr) {
        result = await updateEnrollment(editingEnr.id, efMode, efDiscount, efNote);
        showMessage("Enrollment updated successfully!");
      } else {
        result = await createEnrollment(efStudentId, efCourseId, efMode, efDiscount, efNote);

        const matchesFilters =
          (enrStudentFilter === undefined || enrStudentFilter === result.studentId) &&
          (enrCourseFilter === undefined || enrCourseFilter === result.courseId);

        if (matchesFilters) {
          showMessage(`Enrollment created: ${result.studentName} → ${result.courseName}`);
        } else {
          showMessage(
            `Enrollment created: ${result.studentName} → ${result.courseName}. Clear filters to see it in the list.`
          );
        }
      }

      setEnrModalOpen(false);
      await loadEnrollments();
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  }

  // ---------------- Attendance ----------------
  const [rows, setRows] = useState<Row[]>([]);
  const [loadingAtt, setLoadingAtt] = useState(false);
  const [courseFilter, setCourseFilter] = useState<number | undefined>(undefined);
  const [attQ, setAttQ] = useState("");

  // For search by phone we need students list (shared with invoices and attendance)
  const studentIndex = useMemo(() => {
    const m = new Map<number, StudentDTO>();
    for (const s of students) m.set(s.id, s);
    return m;
  }, [students]);

  const ensureStudentsLoaded = useCallback(async () => {
    if (students.length > 0) return;
    const data = await listStudents("", true);
    setStudents(data);
  }, [students.length]);

  const ensureCoursesLoaded = useCallback(async () => {
    if (courses.length > 0) return;
    const data = await listCourses("");
    setCourses(data);
  }, [courses.length]);

  const loadAttendance = useCallback(async () => {
    setLoadingAtt(true);
    try {
      await ensureStudentsLoaded();
      await ensureCoursesLoaded();
      const data = await fetchRows(year, month, courseFilter);
      setRows(data);
    } finally {
      setLoadingAtt(false);
    }
  }, [year, month, courseFilter, ensureStudentsLoaded, ensureCoursesLoaded]);

  useEffect(() => {
    if (tab === "attendance") loadAttendance();
  }, [tab, loadAttendance]);

  const perLessonTotal = useMemo(() => rows.reduce((s, r) => s + r.count * r.lessonPrice, 0), [rows]);

  const filteredAttendanceRows = useMemo(() => {
    const q = attQ.trim().toLowerCase();
    if (!q) return rows;

    return rows.filter((r) => {
      const s = studentIndex.get(r.studentId);
      const studentName = (r.studentName ?? "").toLowerCase();
      const courseName = (r.courseName ?? "").toLowerCase();
      const phone = (s?.phone ?? "").toLowerCase();
      return studentName.includes(q) || courseName.includes(q) || phone.includes(q);
    });
  }, [rows, attQ, studentIndex]);

  const onChangeCount = async (r: Row, v: number) => {
    if (!Number.isFinite(v)) return;
    const n = v < 0 ? 0 : Math.trunc(v);

    try {
      await saveCount(r.studentId, r.courseId, year, month, n);
      setRows((prev) =>
        prev.map((x) => (x.enrollmentId === r.enrollmentId ? { ...x, count: n } : x))
      );
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onAddAll = async () => {
    try {
      await addOneMass(year, month, courseFilter);
      await loadAttendance();
      showMessage("Added +1 to all visible rows");
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onDeleteEnrollmentFromSheet = async (id: number) => {
    showConfirm(
      "Delete enrollment? This will remove the enrollment and related attendance records. This action cannot be undone.",
      async () => {
        try {
          await deleteEnrollment(id);
          await loadAttendance();
          showMessage("Enrollment deleted successfully!");
        } catch (e: any) {
          showMessage(`Error: ${String(e?.message ?? e)}`, "error");
        }
      }
    );
  };

  // ---------------- Invoices ----------------
  const [invStatus, setInvStatus] = useState<string>("all");
  const [invItems, setInvItems] = useState<InvoiceListItem[]>([]);
  const [selectedInv, setSelectedInv] = useState<InvoiceDTO | null>(null);
  const [loadingInv, setLoadingInv] = useState(false);
  const [invQ, setInvQ] = useState("");
  const [invSummary, setInvSummary] = useState<InvoiceSummaryDTO | null>(null);
  
  // Payment modal state
  const [paymentModalOpen, setPaymentModalOpen] = useState(false);
  const [paymentAmount, setPaymentAmount] = useState("");
  const [paymentMethod, setPaymentMethod] = useState<"cash" | "bank">("cash");
  const [paymentNote, setPaymentNote] = useState("");

  const loadInvoices = useCallback(async () => {
    setLoadingInv(true);
    try {
      await ensureStudentsLoaded();
      const li = await listInvoices(year, month, invStatus);
      setInvItems(li);
      setSelectedInv(null);
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    } finally {
      setLoadingInv(false);
    }
  }, [year, month, invStatus, ensureStudentsLoaded, showMessage]);

  useEffect(() => {
    if (tab === "invoice") loadInvoices();
  }, [tab, loadInvoices]);

  // ---------------- Debtors ----------------
  const [debtors, setDebtors] = useState<DebtorDTO[]>([]);
  const [debtorsLoading, setDebtorsLoading] = useState(false);

  const loadDebtors = useCallback(async () => {
    setDebtorsLoading(true);
    try {
      const data = await listDebtors();
      setDebtors(data);
    } catch (e: any) {
      showMessage(`Error loading debtors: ${String(e?.message ?? e)}`, "error");
    } finally {
      setDebtorsLoading(false);
    }
  }, [showMessage]);

  useEffect(() => {
    if (tab === "debtors") loadDebtors();
  }, [tab, loadDebtors]);

  const filteredInvItems = useMemo(() => {
    const q = invQ.trim().toLowerCase();
    if (!q) return invItems;

    return invItems.filter((it) => {
      const s = studentIndex.get(it.studentId);
      const name = (it.studentName ?? "").toLowerCase();
      const number = (it.number ?? "").toLowerCase();
      const email = (s?.email ?? "").toLowerCase();
      const phone = (s?.phone ?? "").toLowerCase();
      return name.includes(q) || number.includes(q) || email.includes(q) || phone.includes(q);
    });
  }, [invItems, invQ, studentIndex]);

  const onOpenInvoice = async (id: number) => {
    try {
      const iv = await getInvoice(id);
      setSelectedInv(iv);
      // Load payment summary
      if (iv.status !== "draft") {
        const summary = await invoiceSummary(id);
        setInvSummary(summary);
      } else {
        setInvSummary(null);
      }
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  };

  const openPaymentModal = () => {
    if (!selectedInv) return;
    const remaining = invSummary ? invSummary.remaining : selectedInv.total;
    setPaymentAmount(remaining.toFixed(2));
    setPaymentMethod("cash");
    setPaymentNote("");
    setPaymentModalOpen(true);
  };

  const handleCreatePayment = async () => {
    if (!selectedInv) return;
    
    const amount = parseFloat(paymentAmount);
    if (isNaN(amount) || amount <= 0) {
      showMessage("Please enter a valid amount", "error");
      return;
    }

    try {
      const today = new Date().toISOString().split("T")[0]; // YYYY-MM-DD
      await createPayment(
        selectedInv.studentId,
        selectedInv.id,
        amount,
        paymentMethod,
        today,
        paymentNote
      );
      
      setPaymentModalOpen(false);
      showMessage("Payment recorded successfully!");
      
      // Reload invoice and summary
      await onOpenInvoice(selectedInv.id);
      await loadInvoices();
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onGenerateDrafts = async () => {
    try {
      const res = await genDrafts(year, month);
      showMessage(
        `Drafts generated: ${res.created} created, ${res.updated} updated, ${res.skippedHasInvoice} skipped (already issued), ${res.skippedNoLines} skipped (no lines)`
      );
      await loadInvoices();
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onIssueOne = async (id: number) => {
    try {
      const res = await issueOne(id);
      showMessage(`Invoice issued: #${res.number}`);
      await loadInvoices();
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onIssueAll = async () => {
    try {
      const res = await issueAll(year, month);
      showMessage(`Issued: ${res.count}. PDFs: ${res.pdfPaths.length}`);
      await loadInvoices();
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onOpenPdf = async (id: number) => {
    try {
      const path = await ensurePdfAndOpen(id);
      console.log("Opened PDF:", path);
      showMessage("PDF opened");
    } catch (e: any) {
      showMessage(`Error: ${String(e?.message ?? e)}`, "error");
    }
  };

  // ---------------- Render ----------------
  const showMonthPicker = tab === "attendance" || tab === "invoice";

  return (
    <div className="container">
      {/* Global message display */}
      {message && (
        <div
          style={{
            position: "fixed",
            top: "20px",
            right: "20px",
            padding: "16px 24px",
            backgroundColor: message.type === "success" ? "#4caf50" : "#f44336",
            color: "white",
            borderRadius: "4px",
            boxShadow: "0 2px 8px rgba(0,0,0,0.2)",
            zIndex: 10000,
            maxWidth: "400px",
            fontSize: "14px",
            lineHeight: "1.5",
          }}
          role={message.type === "error" ? "alert" : "status"}
          aria-live={message.type === "error" ? "assertive" : "polite"}
          onClick={() => setMessage(null)}
        >
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", gap: "12px" }}>
            <span>{message.text}</span>
            <button
              onClick={(e) => {
                e.stopPropagation();
                setMessage(null);
              }}
              aria-label="Close notification"
              style={{
                background: "none",
                border: "none",
                color: "white",
                cursor: "pointer",
                fontSize: "18px",
                padding: "0",
                lineHeight: "1",
              }}
            >
              ×
            </button>
          </div>
        </div>
      )}

      {/* Global confirmation dialog */}
      {confirmDialog?.isOpen && (
        <div
          style={{
            position: "fixed",
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: "rgba(0, 0, 0, 0.5)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex: 10001,
          }}
        >
          <div
            style={{
              backgroundColor: "white",
              padding: "24px",
              borderRadius: "8px",
              maxWidth: "500px",
              boxShadow: "0 4px 16px rgba(0,0,0,0.3)",
            }}
          >
            <h3 style={{ marginTop: 0, marginBottom: "16px" }}>Confirm Action</h3>
            <p style={{ marginBottom: "24px", lineHeight: "1.5" }}>{confirmDialog.message}</p>
            <div style={{ display: "flex", gap: "12px", justifyContent: "flex-end" }}>
              <button onClick={handleConfirmNo} style={{ padding: "8px 16px" }}>
                Cancel
              </button>
              <button
                onClick={handleConfirmYes}
                style={{
                  padding: "8px 16px",
                  backgroundColor: "#f44336",
                  color: "white",
                  border: "none",
                  borderRadius: "4px",
                  cursor: "pointer",
                }}
              >
                {confirmDialog.confirmButtonLabel ?? "Delete"}
              </button>
            </div>
          </div>
        </div>
      )}

      <nav className="tabs">
        <button className={tab === "students" ? "active" : ""} onClick={() => setTab("students")}>
          Students
        </button>
        <button className={tab === "courses" ? "active" : ""} onClick={() => setTab("courses")}>
          Courses
        </button>
        <button className={tab === "enrollments" ? "active" : ""} onClick={() => setTab("enrollments")}>
          Enrollments
        </button>
        <button className={tab === "attendance" ? "active" : ""} onClick={() => setTab("attendance")}>
          Attendance
        </button>
        <button className={tab === "invoice" ? "active" : ""} onClick={() => setTab("invoice")}>
          Invoices
        </button>
        <button className={tab === "debtors" ? "active" : ""} onClick={() => setTab("debtors")}>
          Debtors
        </button>

        <div className="spacer" />

        {showMonthPicker && (
          <div className="monthpickers">
            <select value={month} onChange={(e) => setMonth(parseInt(e.target.value))}>
              {months.map((m, i) => (
                <option key={m} value={i + 1}>
                  {m}
                </option>
              ))}
            </select>
            <select value={year} onChange={(e) => setYear(parseInt(e.target.value))}>
              {[year - 1, year, year + 1].map((y) => (
                <option key={y} value={y}>
                  {y}
                </option>
              ))}
            </select>
          </div>
        )}
      </nav>

      {/* ---------------- Students ---------------- */}
      {tab === "students" && (
        <>
          <div className="controls">
            <button onClick={openAddStudent}>Add student</button>
            <input
              placeholder="Search name/phone/email…"
              value={studentQ}
              onChange={(e) => setStudentQ(e.target.value)}
              style={{ width: 260 }}
            />
            <label className="inline">
              <input
                type="checkbox"
                checked={includeInactive}
                onChange={(e) => setIncludeInactive(e.target.checked)}
              />
              Include inactive
            </label>
            <button onClick={loadStudents}>Refresh</button>
          </div>

          {studentLoading ? (
            <div>Loading…</div>
          ) : students.length === 0 ? (
            <div className="empty">No students yet.</div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Phone</th>
                  <th>Email</th>
                  <th>Active</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {students.map((s) => (
                  <tr key={s.id}>
                    <td>{s.fullName}</td>
                    <td>{s.phone}</td>
                    <td>{s.email}</td>
                    <td>{s.isActive ? "yes" : "no"}</td>
                    <td>
                      <button onClick={() => openEditStudent(s)}>Edit</button>
                      <button onClick={() => toggleStudentActive(s)}>{s.isActive ? "Deactivate" : "Activate"}</button>
                      {!s.isActive && <button onClick={() => removeStudent(s.id)}>Delete</button>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {studentModalOpen && (
            <div className="modal">
              <div className="modalBody">
                <h3>{editingStudent ? "Edit student" : "Add student"}</h3>

                <div className="formRow">
                  <label>Full name</label>
                  <input value={sfName} onChange={(e) => setSfName(e.target.value)} />
                </div>
                <div className="formRow">
                  <label>Phone</label>
                  <input value={sfPhone} onChange={(e) => setSfPhone(e.target.value)} />
                </div>
                <div className="formRow">
                  <label>Email</label>
                  <input value={sfEmail} onChange={(e) => setSfEmail(e.target.value)} />
                </div>
                <div className="formRow">
                  <label>Note</label>
                  <input value={sfNote} onChange={(e) => setSfNote(e.target.value)} />
                </div>

                <div className="modalActions">
                  <button onClick={saveStudent}>Save</button>
                  <button onClick={() => setStudentModalOpen(false)}>Cancel</button>
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {/* ---------------- Courses ---------------- */}
      {tab === "courses" && (
        <>
          <div className="controls">
            <button onClick={openAddCourse}>Add course</button>
            <input
              placeholder="Search course name…"
              value={courseQ}
              onChange={(e) => setCourseQ(e.target.value)}
              style={{ width: 260 }}
            />
            <button onClick={loadCourses}>Refresh</button>
          </div>

          {courseLoading ? (
            <div>Loading…</div>
          ) : courses.length === 0 ? (
            <div className="empty">No courses yet.</div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Type</th>
                  <th style={{ textAlign: "right" }}>Lesson</th>
                  <th style={{ textAlign: "right" }}>Subscription</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {courses.map((c) => (
                  <tr key={c.id}>
                    <td>{c.name}</td>
                    <td>{c.type}</td>
                    <td style={{ textAlign: "right" }}>{c.lessonPrice.toFixed(2)}</td>
                    <td style={{ textAlign: "right" }}>{c.subscriptionPrice.toFixed(2)}</td>
                    <td>
                      <button onClick={() => openEditCourse(c)}>Edit</button>
                      <button onClick={() => removeCourse(c.id)}>Delete</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {courseModalOpen && (
            <div className="modal">
              <div className="modalBody">
                <h3>{editingCourse ? "Edit course" : "Add course"}</h3>

                <div className="formRow">
                  <label>Name</label>
                  <input value={cfName} onChange={(e) => setCfName(e.target.value)} />
                </div>

                <div className="formRow">
                  <label>Type</label>
                  <select value={cfType} onChange={(e) => setCfType(e.target.value as any)}>
                    <option value="group">group</option>
                    <option value="individual">individual</option>
                  </select>
                </div>

                <div className="formRow">
                  <label>Lesson price</label>
                  <input
                    type="number"
                    min={0}
                    step="0.01"
                    value={cfLessonPrice}
                    onChange={(e) => setCfLessonPrice(numOrZero(e.target.value))}
                  />
                </div>

                <div className="formRow">
                  <label>Subscription price</label>
                  <input
                    type="number"
                    min={0}
                    step="0.01"
                    value={cfSubscriptionPrice}
                    onChange={(e) => setCfSubscriptionPrice(numOrZero(e.target.value))}
                  />
                </div>

                <div className="modalActions">
                  <button onClick={saveCourse}>Save</button>
                  <button onClick={() => setCourseModalOpen(false)}>Cancel</button>
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {/* ---------------- Enrollments ---------------- */}
      {tab === "enrollments" && (
        <>
          <div className="controls">
            <button onClick={openAddEnrollment}>Add enrollment</button>

            <select value={enrStudentFilter ?? ""} onChange={(e) => setEnrStudentFilter(intOrUndef(e.target.value))}>
              <option value="">All students</option>
              {students.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.fullName}
                </option>
              ))}
            </select>

            <select value={enrCourseFilter ?? ""} onChange={(e) => setEnrCourseFilter(intOrUndef(e.target.value))}>
              <option value="">All courses</option>
              {courses.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>

            <button onClick={loadEnrollments}>Refresh</button>
          </div>

          {enrLoading ? (
            <div>Loading…</div>
          ) : enrollments.length === 0 ? (
            <div className="empty">No enrollments yet.</div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Student</th>
                  <th>Course</th>
                  <th>Billing</th>
                  <th style={{ textAlign: "right" }}>Discount</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {enrollments.map((e) => (
                  <tr key={e.id}>
                    <td>{e.studentName}</td>
                    <td>{e.courseName}</td>
                    <td>{e.billingMode}</td>
                    <td style={{ textAlign: "right" }}>{e.discountPct.toFixed(1)}%</td>
                    <td>
                      <button onClick={() => openEditEnrollment(e)}>Edit</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {enrModalOpen && (
            <div className="modal">
              <div className="modalBody">
                <h3>{editingEnr ? "Edit enrollment" : "Add enrollment"}</h3>

                <div className="formRow">
                  <label>Student</label>
                  <select
                    value={efStudentId}
                    disabled={!!editingEnr}
                    onChange={(e) => setEfStudentId(parseInt(e.target.value))}
                  >
                    {students
                      .filter((s) => s.isActive)
                      .map((s) => (
                        <option key={s.id} value={s.id}>
                          {s.fullName}
                        </option>
                      ))}
                  </select>
                </div>

                <div className="formRow">
                  <label>Course</label>
                  <select value={efCourseId} disabled={!!editingEnr} onChange={(e) => setEfCourseId(parseInt(e.target.value))}>
                    {courses.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="formRow">
                  <label>Billing</label>
                  <select value={efMode} onChange={(e) => setEfMode(e.target.value as any)}>
                    <option value="per_lesson">per_lesson</option>
                    <option value="subscription">subscription</option>
                  </select>
                </div>

                <div className="formRow">
                  <label>Discount %</label>
                  <input
                    type="number"
                    min={0}
                    max={100}
                    step="0.1"
                    value={efDiscount}
                    onChange={(e) => setEfDiscount(numOrZero(e.target.value))}
                  />
                </div>

                <div className="formRow">
                  <label>Note</label>
                  <input value={efNote} onChange={(e) => setEfNote(e.target.value)} />
                </div>

                <div className="modalActions">
                  <button onClick={saveEnrollment}>Save</button>
                  <button onClick={() => setEnrModalOpen(false)}>Cancel</button>
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {/* ---------------- Attendance ---------------- */}
      {tab === "attendance" && (
        <>
          <div className="controls">
            <button onClick={onAddAll}>+1 all</button>

            <select value={courseFilter ?? ""} onChange={(e) => setCourseFilter(intOrUndef(e.target.value))}>
              <option value="">All groups</option>
              {courses.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>

            <input
              placeholder="Search student / phone / group…"
              value={attQ}
              onChange={(e) => setAttQ(e.target.value)}
              style={{ width: 260 }}
            />

            <button onClick={loadAttendance}>Refresh</button>
          </div>

          {loadingAtt ? (
            <div>Loading…</div>
          ) : filteredAttendanceRows.length === 0 ? (
            <div className="empty">
              {attQ.trim() ? "No matches found for your search." : "No per-lesson rows. Create enrollments first."}
            </div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Student</th>
                  <th>Course</th>
                  <th style={{ textAlign: "right" }}>Lesson price</th>
                  <th style={{ textAlign: "right" }}>Count</th>
                  <th style={{ textAlign: "right" }}>Total</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {filteredAttendanceRows.map((r) => (
                  <tr key={r.enrollmentId}>
                    <td>{r.studentName}</td>
                    <td>
                      {r.courseName} ({r.courseType})
                    </td>
                    <td style={{ textAlign: "right" }}>{r.lessonPrice.toFixed(2)}</td>
                    <td style={{ textAlign: "right" }}>
                      <input
                        type="number"
                        min={0}
                        value={r.count}
                        onChange={(e) => onChangeCount(r, Number(e.target.value))}
                        style={{ width: "5rem", textAlign: "right" }}
                      />
                    </td>
                    <td style={{ textAlign: "right" }}>{(r.count * r.lessonPrice).toFixed(2)}</td>
                    <td>
                      <button onClick={() => onDeleteEnrollmentFromSheet(r.enrollmentId)}>Delete enrollment</button>
                    </td>
                  </tr>
                ))}
              </tbody>
              <tfoot>
                <tr>
                  <td colSpan={4} style={{ textAlign: "right" }}>
                    Per-lesson total:
                  </td>
                  <td style={{ textAlign: "right" }}>{perLessonTotal.toFixed(2)}</td>
                  <td></td>
                </tr>
              </tfoot>
            </table>
          )}
        </>
      )}

      {/* ---------------- Invoices ---------------- */}
      {tab === "invoice" && (
        <>
          <div className="controls">
            <button onClick={onGenerateDrafts}>Generate drafts</button>
            <button onClick={onIssueAll}>Issue all</button>

            <select value={invStatus} onChange={(e) => setInvStatus(e.target.value)}>
              <option value="draft">draft</option>
              <option value="issued">issued</option>
              <option value="paid">paid</option>
              <option value="all">all</option>
            </select>

            <input
              placeholder="Search student / phone / email / invoice number"
              value={invQ}
              onChange={(e) => setInvQ(e.target.value)}
              style={{ width: "320px" }}
            />

            <button onClick={loadInvoices}>Refresh</button>
          </div>

          {loadingInv ? (
            <div>Loading…</div>
          ) : filteredInvItems.length === 0 ? (
            <div className="empty">No invoices found for this period/status/search.</div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Student</th>
                  <th>Period</th>
                  <th style={{ textAlign: "right" }}>Total</th>
                  <th>Status</th>
                  <th>Number</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {filteredInvItems.map((it) => (
                  <tr key={it.id}>
                    <td>{it.studentName}</td>
                    <td>
                      {months[it.month - 1]} {it.year}
                    </td>
                    <td style={{ textAlign: "right" }}>{it.total.toFixed(2)}</td>
                    <td>{it.status}</td>
                    <td>{it.number ?? ""}</td>
                    <td>
                      <button onClick={() => onOpenInvoice(it.id)}>Open</button>
                      {it.status === "draft" && <button onClick={() => onIssueOne(it.id)}>Issue</button>}
                      {it.status !== "draft" && <button onClick={() => onOpenPdf(it.id)}>PDF</button>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {selectedInv && (
            <div className="panel">
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1rem" }}>
                <h3>
                  Invoice {selectedInv.number ? `#${selectedInv.number}` : ""} — {selectedInv.studentName} —{" "}
                  {months[selectedInv.month - 1]} {selectedInv.year}
                </h3>
                {selectedInv.status !== "draft" && (
                  <button onClick={openPaymentModal}>Record Payment</button>
                )}
              </div>

              {invSummary && selectedInv.status !== "draft" && (
                <div style={{ marginBottom: "1rem", padding: "0.5rem", backgroundColor: "#f5f5f5", borderRadius: "4px" }}>
                  <div style={{ display: "flex", justifyContent: "space-between" }}>
                    <span>Total:</span>
                    <span style={{ fontWeight: "bold" }}>€{invSummary.total.toFixed(2)}</span>
                  </div>
                  <div style={{ display: "flex", justifyContent: "space-between" }}>
                    <span>Paid:</span>
                    <span style={{ color: "#2e7d32" }}>€{invSummary.paid.toFixed(2)}</span>
                  </div>
                  <div style={{ display: "flex", justifyContent: "space-between" }}>
                    <span>Remaining:</span>
                    <span style={{ fontWeight: "bold", color: invSummary.remaining > 0 ? "#d32f2f" : "#2e7d32" }}>
                      €{invSummary.remaining.toFixed(2)}
                    </span>
                  </div>
                  <div style={{ display: "flex", justifyContent: "space-between", marginTop: "0.5rem" }}>
                    <span>Status:</span>
                    <span style={{ fontWeight: "bold" }}>{invSummary.status}</span>
                  </div>
                </div>
              )}

              <table>
                <thead>
                  <tr>
                    <th>Description</th>
                    <th style={{ textAlign: "right" }}>Qty</th>
                    <th style={{ textAlign: "right" }}>Unit</th>
                    <th style={{ textAlign: "right" }}>Amount</th>
                  </tr>
                </thead>
                <tbody>
                  {selectedInv.lines.map((l, idx) => (
                    <tr key={idx}>
                      <td>{l.description}</td>
                      <td style={{ textAlign: "right" }}>{l.qty}</td>
                      <td style={{ textAlign: "right" }}>{l.unitPrice.toFixed(2)}</td>
                      <td style={{ textAlign: "right" }}>{l.amount.toFixed(2)}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr>
                    <td colSpan={3} style={{ textAlign: "right" }}>
                      Total:
                    </td>
                    <td style={{ textAlign: "right" }}>{selectedInv.total.toFixed(2)}</td>
                  </tr>
                </tfoot>
              </table>
            </div>
          )}

          {/* Payment Modal */}
          {paymentModalOpen && selectedInv && (
            <div className="modal" onClick={() => setPaymentModalOpen(false)}>
              <div className="modalBody" onClick={(e) => e.stopPropagation()}>
                <h3>Record Payment</h3>
                <div className="formRow">
                  <label>Amount (€):</label>
                  <input
                    type="number"
                    step="0.01"
                    value={paymentAmount}
                    onChange={(e) => setPaymentAmount(e.target.value)}
                    autoFocus
                  />
                </div>
                <div className="formRow">
                  <label>Method:</label>
                  <select
                    value={paymentMethod}
                    onChange={(e) => setPaymentMethod(e.target.value as "cash" | "bank")}
                  >
                    <option value="cash">Cash</option>
                    <option value="bank">Bank</option>
                  </select>
                </div>
                <div className="formRow">
                  <label>Note (optional):</label>
                  <input
                    type="text"
                    value={paymentNote}
                    onChange={(e) => setPaymentNote(e.target.value)}
                    placeholder="Payment note..."
                  />
                </div>
                <div className="modalActions">
                  <button onClick={() => setPaymentModalOpen(false)}>Cancel</button>
                  <button onClick={handleCreatePayment}>Record Payment</button>
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {/* ---------------- Debtors ---------------- */}
      {tab === "debtors" && (
        <>
          <div className="controls">
            <button onClick={loadDebtors}>Refresh</button>
          </div>

          {debtorsLoading ? (
            <div>Loading…</div>
          ) : debtors.length === 0 ? (
            <div className="empty">No debtors found. All students are up to date with payments.</div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Student Name</th>
                  <th style={{ textAlign: "right" }}>Debt</th>
                  <th style={{ textAlign: "right" }}>Total Invoiced</th>
                  <th style={{ textAlign: "right" }}>Total Paid</th>
                </tr>
              </thead>
              <tbody>
                {debtors.map((d) => (
                  <tr key={d.studentId}>
                    <td>{d.studentName}</td>
                    <td style={{ textAlign: "right", fontWeight: "bold", color: "#d32f2f" }}>
                      €{d.debt.toFixed(2)}
                    </td>
                    <td style={{ textAlign: "right" }}>€{d.totalInvoiced.toFixed(2)}</td>
                    <td style={{ textAlign: "right" }}>€{d.totalPaid.toFixed(2)}</td>
                  </tr>
                ))}
              </tbody>
              <tfoot>
                <tr>
                  <td style={{ fontWeight: "bold" }}>Total Debt:</td>
                  <td style={{ textAlign: "right", fontWeight: "bold", color: "#d32f2f" }}>
                    €{debtors.reduce((sum, d) => sum + d.debt, 0).toFixed(2)}
                  </td>
                  <td colSpan={2}></td>
                </tr>
              </tfoot>
            </table>
          )}
        </>
      )}
    </div>
  );
}
