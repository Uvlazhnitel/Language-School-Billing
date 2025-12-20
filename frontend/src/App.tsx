import { useEffect, useMemo, useState, useCallback } from "react";
import "./App.css";

import {
  fetchRows, saveCount, addOneMass, estimateBySchedule, setLocked,
  devSeed, devReset, deleteEnrollment, Row
} from "./lib/attendance";

import {
  genDrafts, listInvoices, getInvoice, deleteDraft,
  issueOne, issueAll, ensurePdfAndOpen,
  InvoiceListItem, InvoiceDTO
} from "./lib/invoices";


import { listStudents, createStudent, updateStudent, setStudentActive, StudentDTO } from "./lib/students";
import { listCourses, createCourse, updateCourse, deleteCourse, CourseDTO } from "./lib/courses";
import { listEnrollments, createEnrollment, updateEnrollment, endEnrollment, EnrollmentDTO } from "./lib/enrollments";

const months = ["January","February","March","April","May","June","July","August","September","October","November","December"];

type Tab = "students" | "courses" | "enrollments" | "attendance" | "invoice";

function todayYMD() {
  const d = new Date();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${mm}-${dd}`;
}

const weekdayLabels: { value: number; label: string }[] = [
  { value: 1, label: "Mon" },
  { value: 2, label: "Tue" },
  { value: 3, label: "Wed" },
  { value: 4, label: "Thu" },
  { value: 5, label: "Fri" },
  { value: 6, label: "Sat" },
  { value: 0, label: "Sun" },
];

// Helper: parse string to number, return 0 for empty/invalid
function numOrZero(s: string): number {
  if (s.trim() === "") return 0;
  const n = Number(s);
  return Number.isFinite(n) ? n : 0;
}

// Helper: parse string to int or undefined for filter selects
function intOrUndef(s: string): number | undefined {
  if (s.trim() === "") return undefined;
  const n = Number(s);
  return Number.isFinite(n) ? Math.trunc(n) : undefined;
}

export default function App() {
  const now = new Date();
  const [tab, setTab] = useState<Tab>("students");

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
      alert("Full name is required");
      return;
    }
    if (editingStudent) {
      await updateStudent(editingStudent.id, sfName, sfPhone, sfEmail, sfNote);
    } else {
      await createStudent(sfName, sfPhone, sfEmail, sfNote);
    }
    setStudentModalOpen(false);
    await loadStudents();
  }

  async function toggleStudentActive(s: StudentDTO) {
    await setStudentActive(s.id, !s.isActive);
    await loadStudents();
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
  const [cfDays, setCfDays] = useState<number[]>([]);

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
    setCfDays([]);
    setCourseModalOpen(true);
  }

  function openEditCourse(c: CourseDTO) {
    setEditingCourse(c);
    setCfName(c.name);
    setCfType(c.type);
    setCfLessonPrice(c.lessonPrice);
    setCfSubscriptionPrice(c.subscriptionPrice);
    setCfDays(c.scheduleDays || []);
    setCourseModalOpen(true);
  }

  function toggleDay(d: number) {
    setCfDays(prev => prev.includes(d) ? prev.filter(x => x !== d) : [...prev, d]);
  }

  async function saveCourse() {
    if (!cfName.trim()) {
      alert("Course name is required");
      return;
    }
    if (cfLessonPrice < 0 || cfSubscriptionPrice < 0) {
      alert("Prices must be >= 0");
      return;
    }
    const days = cfType === "group" ? cfDays : [];
    if (editingCourse) {
      await updateCourse(editingCourse.id, cfName, cfType, cfLessonPrice, cfSubscriptionPrice, days);
    } else {
      await createCourse(cfName, cfType, cfLessonPrice, cfSubscriptionPrice, days);
    }
    setCourseModalOpen(false);
    await loadCourses();
  }

  async function removeCourse(id: number) {
    if (!confirm("Delete course? This is blocked if enrollments exist.")) return;
    try {
      await deleteCourse(id);
      await loadCourses();
    } catch (e: any) {
      alert(String(e?.message ?? e));
    }
  }

  // ---------------- Enrollments ----------------
  const [enrollments, setEnrollments] = useState<EnrollmentDTO[]>([]);
  const [enrActiveOnly, setEnrActiveOnly] = useState(true);
  const [enrStudentFilter, setEnrStudentFilter] = useState<number | undefined>(undefined);
  const [enrCourseFilter, setEnrCourseFilter] = useState<number | undefined>(undefined);
  const [enrLoading, setEnrLoading] = useState(false);

  const [enrModalOpen, setEnrModalOpen] = useState(false);
  const [editingEnr, setEditingEnr] = useState<EnrollmentDTO | null>(null);
  const [efStudentId, setEfStudentId] = useState<number>(0);
  const [efCourseId, setEfCourseId] = useState<number>(0);
  const [efMode, setEfMode] = useState<"subscription" | "per_lesson">("per_lesson");
  const [efStart, setEfStart] = useState(todayYMD());
  const [efEnd, setEfEnd] = useState<string>("");
  const [efDiscount, setEfDiscount] = useState(0);
  const [efNote, setEfNote] = useState("");

  const loadEnrollments = useCallback(async () => {
    setEnrLoading(true);
    try {
      // Load students & courses if needed for dropdowns
      await Promise.all([
        students.length === 0 ? listStudents("", true).then(setStudents) : Promise.resolve(),
        courses.length === 0 ? listCourses("").then(setCourses) : Promise.resolve()
      ]);
      const data = await listEnrollments(enrStudentFilter, enrCourseFilter, enrActiveOnly);
      setEnrollments(data);
    } finally {
      setEnrLoading(false);
    }
  }, [enrStudentFilter, enrCourseFilter, enrActiveOnly, students.length, courses.length]);

  useEffect(() => {
    if (tab === "enrollments") loadEnrollments();
  }, [tab, loadEnrollments]);

  function openAddEnrollment() {
    // Guard: check if students and courses exist
    if (students.length === 0) {
      alert("No students available. Please add students first.");
      setTab("students");
      return;
    }
    if (courses.length === 0) {
      alert("No courses available. Please add courses first.");
      setTab("courses");
      return;
    }
    setEditingEnr(null);
    setEfStudentId(students[0]?.id ?? 0);
    setEfCourseId(courses[0]?.id ?? 0);
    setEfMode("per_lesson");
    setEfStart(todayYMD());
    setEfEnd("");
    setEfDiscount(0);
    setEfNote("");
    setEnrModalOpen(true);
  }

  function openEditEnrollment(e: EnrollmentDTO) {
    setEditingEnr(e);
    setEfStudentId(e.studentId);
    setEfCourseId(e.courseId);
    setEfMode(e.billingMode);
    setEfStart(e.startDate);
    setEfEnd(e.endDate ?? "");
    setEfDiscount(e.discountPct);
    setEfNote(e.note);
    setEnrModalOpen(true);
  }

  async function saveEnrollment() {
    if (efStudentId <= 0 || efCourseId <= 0) {
      alert("Select student and course");
      return;
    }
    if (efDiscount < 0 || efDiscount > 100) {
      alert("Discount must be 0..100");
      return;
    }
    if (editingEnr) {
      await updateEnrollment(editingEnr.id, efMode, efStart, efEnd ? efEnd : undefined, efDiscount, efNote);
    } else {
      await createEnrollment(efStudentId, efCourseId, efMode, efStart, efEnd ? efEnd : undefined, efDiscount, efNote);
    }
    setEnrModalOpen(false);
    await loadEnrollments();
  }

  async function endThisEnrollment(id: number) {
    const d = prompt("End date (YYYY-MM-DD)", todayYMD());
    if (!d) return;
    await endEnrollment(id, d);
    await loadEnrollments();
  }

  // ---------------- Attendance (existing) ----------------
  const [rows, setRows] = useState<Row[]>([]);
  const [loadingAtt, setLoadingAtt] = useState(false);
  const [courseFilter, setCourseFilter] = useState<number | undefined>(undefined);
  const [msg, setMsg] = useState("");

  const loadAttendance = useCallback(async () => {
    setLoadingAtt(true);
    try {
      const data = await fetchRows(year, month, courseFilter);
      setRows(data);
    } finally {
      setLoadingAtt(false);
    }
  }, [year, month, courseFilter]);

  useEffect(() => {
    if (tab === "attendance") loadAttendance();
  }, [tab, loadAttendance]);

  const perLessonTotal = useMemo(
    () => rows.reduce((s, r) => s + r.count * r.lessonPrice, 0),
    [rows]
  );

  const onChangeCount = async (r: Row, v: number) => {
    if (!Number.isFinite(v)) return; // keep previous when empty/invalid
    const n = v < 0 ? 0 : Math.trunc(v);
    await saveCount(r.studentId, r.courseId, year, month, n);
    setRows(rows.map(x => (x.enrollmentId === r.enrollmentId ? { ...x, count: n } : x)));
  };

  const onAddAll = async () => { await addOneMass(year, month, courseFilter); await loadAttendance(); };
  const onEstimate = async () => {
    const hints = await estimateBySchedule(year, month, courseFilter);
    const patched = rows.map(r => {
      const key = `${r.studentId}-${r.courseId}`;
      const hint = hints[key] ?? 0;
      return r.count === 0 && hint > 0 ? { ...r, count: hint } : r;
    });
    setRows(patched);
    for (const r of patched) {
      const prev = rows.find(x => x.enrollmentId === r.enrollmentId)?.count ?? 0;
      if (r.count !== prev) await saveCount(r.studentId, r.courseId, year, month, r.count);
    }
    setMsg("Schedule hint applied"); setTimeout(() => setMsg(""), 1500);
  };
  const onLock = async (lock: boolean) => { await setLocked(year, month, courseFilter, lock); await loadAttendance(); };

  // You can keep dev tools during development, and hide them for defense later.
  const onSeed = async () => { await devSeed(); await loadAttendance(); };
  const onReset = async () => { await devReset(); await loadAttendance(); };
  const onDeleteEnrollmentFromSheet = async (id: number) => { await deleteEnrollment(id); await loadAttendance(); };

  // ---------------- Invoices ----------------
  const [invStatus, setInvStatus] = useState<string>("draft");
  const [invItems, setInvItems] = useState<InvoiceListItem[]>([]);
  const [selectedInv, setSelectedInv] = useState<InvoiceDTO | null>(null);
  const [loadingInv, setLoadingInv] = useState(false);

  const loadInvoices = useCallback(async () => {
    setLoadingInv(true);
    try {
      const li = await listInvoices(year, month, invStatus);
      setInvItems(li);
      setSelectedInv(null);
    } finally {
      setLoadingInv(false);
    }
  }, [year, month, invStatus]);

  useEffect(() => {
    if (tab === "invoice") loadInvoices();
  }, [tab, loadInvoices]);

  const onGenDrafts = async () => {
    const res = await genDrafts(year, month);
    alert(`Drafts: created=${res.created}, updated=${res.updated}, skippedHasInvoice=${res.skippedHasInvoice}, skippedNoLines=${res.skippedNoLines}`);
    await loadInvoices();
  };

  const onOpenInvoice = async (id: number) => {
    const iv = await getInvoice(id);
    setSelectedInv(iv);
  };

  const onDeleteDraft = async (id: number) => {
    await deleteDraft(id);
    await loadInvoices();
  };

  const onIssueOne = async (id: number) => {
    const res = await issueOne(id);
    alert(`Issued: ${res.number}\nPDF: ${res.pdfPath}`);
    await loadInvoices();
  };

  const onIssueAll = async () => {
    const res = await issueAll(year, month);
    alert(`Issued: ${res.count}\nPDF:\n${res.pdfPaths.join("\n")}`);
    await loadInvoices();
  };

  const onOpenPdf = async (id: number) => {
    const path = await ensurePdfAndOpen(id);
    console.log("Opened PDF:", path);
  };

  // ---------------- Render ----------------
  const showMonthPicker = tab === "attendance" || tab === "invoice";

  return (
    <div className="container">
      <nav className="tabs">
        <button className={tab === "students" ? "active" : ""} onClick={() => setTab("students")}>Students</button>
        <button className={tab === "courses" ? "active" : ""} onClick={() => setTab("courses")}>Courses</button>
        <button className={tab === "enrollments" ? "active" : ""} onClick={() => setTab("enrollments")}>Enrollments</button>
        <button className={tab === "attendance" ? "active" : ""} onClick={() => setTab("attendance")}>Attendance</button>
        <button className={tab === "invoice" ? "active" : ""} onClick={() => setTab("invoice")}>Invoices</button>

        <div className="spacer" />

        {showMonthPicker && (
          <div className="monthpickers">
            <select value={month} onChange={(e) => setMonth(parseInt(e.target.value))}>
              {months.map((m, i) => (<option key={m} value={i + 1}>{m}</option>))}
            </select>
            <select value={year} onChange={(e) => setYear(parseInt(e.target.value))}>
              {[year - 1, year, year + 1].map(y => (<option key={y} value={y}>{y}</option>))}
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
              <input type="checkbox" checked={includeInactive} onChange={(e) => setIncludeInactive(e.target.checked)} />
              Include inactive
            </label>
            <button onClick={loadStudents}>Refresh</button>
          </div>

          {studentLoading ? <div>Loading…</div> : (
            students.length === 0 ? <div className="empty">No students yet.</div> : (
              <table>
                <thead>
                  <tr>
                    <th>Name</th><th>Phone</th><th>Email</th><th>Active</th><th></th>
                  </tr>
                </thead>
                <tbody>
                  {students.map(s => (
                    <tr key={s.id}>
                      <td>{s.fullName}</td>
                      <td>{s.phone}</td>
                      <td>{s.email}</td>
                      <td>{s.isActive ? "yes" : "no"}</td>
                      <td>
                        <button onClick={() => openEditStudent(s)}>Edit</button>
                        <button onClick={() => toggleStudentActive(s)}>
                          {s.isActive ? "Deactivate" : "Activate"}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )
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

          {courseLoading ? <div>Loading…</div> : (
            courses.length === 0 ? <div className="empty">No courses yet.</div> : (
              <table>
                <thead>
                  <tr>
                    <th>Name</th><th>Type</th><th style={{textAlign:"right"}}>Lesson</th><th style={{textAlign:"right"}}>Subscription</th><th>Schedule</th><th></th>
                  </tr>
                </thead>
                <tbody>
                  {courses.map(c => (
                    <tr key={c.id}>
                      <td>{c.name}</td>
                      <td>{c.type}</td>
                      <td style={{textAlign:"right"}}>{c.lessonPrice.toFixed(2)}</td>
                      <td style={{textAlign:"right"}}>{c.subscriptionPrice.toFixed(2)}</td>
                      <td>{(c.scheduleDays || []).join(", ")}</td>
                      <td>
                        <button onClick={() => openEditCourse(c)}>Edit</button>
                        <button onClick={() => removeCourse(c.id)}>Delete</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )
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
                  <input type="number" min={0} step="0.01" value={cfLessonPrice}
                         onChange={(e) => setCfLessonPrice(numOrZero(e.target.value))} />
                </div>

                <div className="formRow">
                  <label>Subscription price</label>
                  <input type="number" min={0} step="0.01" value={cfSubscriptionPrice}
                         onChange={(e) => setCfSubscriptionPrice(numOrZero(e.target.value))} />
                </div>

                {cfType === "group" && (
                  <div className="formRow">
                    <label>Schedule</label>
                    <div className="days">
                      {weekdayLabels.map(d => (
                        <label key={d.value} className="inline">
                          <input
                            type="checkbox"
                            checked={cfDays.includes(d.value)}
                            onChange={() => toggleDay(d.value)}
                          />
                          {d.label}
                        </label>
                      ))}
                    </div>
                  </div>
                )}

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

            <label className="inline">
              Active only
              <input type="checkbox" checked={enrActiveOnly} onChange={(e) => setEnrActiveOnly(e.target.checked)} />
            </label>

            <select value={enrStudentFilter ?? ""} onChange={(e) => setEnrStudentFilter(intOrUndef(e.target.value))}>
              <option value="">All students</option>
              {students.map(s => <option key={s.id} value={s.id}>{s.fullName}</option>)}
            </select>

            <select value={enrCourseFilter ?? ""} onChange={(e) => setEnrCourseFilter(intOrUndef(e.target.value))}>
              <option value="">All courses</option>
              {courses.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>

            <button onClick={loadEnrollments}>Refresh</button>
          </div>

          {enrLoading ? <div>Loading…</div> : (
            enrollments.length === 0 ? <div className="empty">No enrollments yet.</div> : (
              <table>
                <thead>
                  <tr>
                    <th>Student</th><th>Course</th><th>Billing</th><th>Start</th><th>End</th><th style={{textAlign:"right"}}>Discount</th><th></th>
                  </tr>
                </thead>
                <tbody>
                  {enrollments.map(e => (
                    <tr key={e.id}>
                      <td>{e.studentName}</td>
                      <td>{e.courseName}</td>
                      <td>{e.billingMode}</td>
                      <td>{e.startDate}</td>
                      <td>{e.endDate ?? ""}</td>
                      <td style={{textAlign:"right"}}>{e.discountPct.toFixed(1)}%</td>
                      <td>
                        <button onClick={() => openEditEnrollment(e)}>Edit</button>
                        <button onClick={() => endThisEnrollment(e.id)}>End</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )
          )}

          {enrModalOpen && (
            <div className="modal">
              <div className="modalBody">
                <h3>{editingEnr ? "Edit enrollment" : "Add enrollment"}</h3>

                <div className="formRow">
                  <label>Student</label>
                  <select value={efStudentId} disabled={!!editingEnr}
                          onChange={(e) => setEfStudentId(parseInt(e.target.value))}>
                    {students.map(s => <option key={s.id} value={s.id}>{s.fullName}</option>)}
                  </select>
                </div>

                <div className="formRow">
                  <label>Course</label>
                  <select value={efCourseId} disabled={!!editingEnr}
                          onChange={(e) => setEfCourseId(parseInt(e.target.value))}>
                    {courses.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
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
                  <label>Start date</label>
                  <input type="date" value={efStart} onChange={(e) => setEfStart(e.target.value)} />
                </div>

                <div className="formRow">
                  <label>End date</label>
                  <input type="date" value={efEnd} onChange={(e) => setEfEnd(e.target.value)} />
                </div>

                <div className="formRow">
                  <label>Discount %</label>
                  <input type="number" min={0} max={100} step="0.1" value={efDiscount}
                         onChange={(e) => setEfDiscount(numOrZero(e.target.value))} />
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
          {msg && <div className="msg">{msg}</div>}
          <div className="controls">
            <button onClick={onEstimate}>Schedule hint</button>
            <button onClick={onAddAll}>+1 all</button>
            <button onClick={() => onLock(true)}>Lock month</button>
            <button onClick={() => onLock(false)}>Unlock</button>

            <div className="spacer" />

            {/* Dev tools (optional) */}
            <button onClick={onSeed}>Seed demo</button>
            <button onClick={onReset}>Reset demo</button>
          </div>

          {loadingAtt ? <div>Loading…</div> : (
            rows.length === 0 ? <div className="empty">No per-lesson rows. Create enrollments first.</div> :
              <table>
                <thead>
                  <tr>
                    <th>Student</th><th>Course</th><th style={{textAlign:"right"}}>Lesson price</th><th style={{textAlign:"right"}}>Count</th><th style={{textAlign:"right"}}>Total</th><th>Lock</th><th></th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map(r => (
                    <tr key={r.enrollmentId}>
                      <td>{r.studentName}</td>
                      <td>{r.courseName} ({r.courseType})</td>
                      <td style={{ textAlign: "right" }}>{r.lessonPrice.toFixed(2)}</td>
                      <td style={{ textAlign: "right" }}>
                        <input
                          type="number"
                          min={0}
                          value={r.count}
                          disabled={r.locked}
                          onChange={(e) => onChangeCount(r, Number(e.target.value))}
                          style={{ width: "5rem", textAlign: "right" }}
                        />
                      </td>
                      <td style={{ textAlign: "right" }}>{(r.count * r.lessonPrice).toFixed(2)}</td>
                      <td>{r.locked ? "locked" : "open"}</td>
                      <td><button onClick={() => onDeleteEnrollmentFromSheet(r.enrollmentId)}>Delete enrollment (danger)</button></td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr>
                    <td colSpan={4} style={{ textAlign: "right" }}>Per-lesson total:</td>
                    <td style={{ textAlign: "right" }}>{perLessonTotal.toFixed(2)}</td>
                    <td colSpan={2}></td>
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
            <button onClick={onGenDrafts}>Build drafts</button>
            <button onClick={onIssueAll}>Issue all</button>
            <select value={invStatus} onChange={(e) => setInvStatus(e.target.value)}>
              <option value="draft">draft</option>
              <option value="issued">issued</option>
              <option value="paid">paid</option>
              <option value="all">all</option>
            </select>
            <button onClick={loadInvoices}>Refresh</button>
          </div>

          {loadingInv ? <div>Loading…</div> : (
            invItems.length === 0 ? <div className="empty">No invoices for this period/status.</div> :
              <table>
                <thead>
                  <tr>
                    <th>Student</th><th>Period</th><th style={{textAlign:"right"}}>Lines</th><th style={{textAlign:"right"}}>Total</th><th>Status</th><th>Number</th><th></th>
                  </tr>
                </thead>
                <tbody>
                  {invItems.map(it => (
                    <tr key={it.id}>
                      <td>{it.studentName}</td>
                      <td>{months[it.month - 1]} {it.year}</td>
                      <td style={{ textAlign: "right" }}>{it.linesCount}</td>
                      <td style={{ textAlign: "right" }}>{it.total.toFixed(2)}</td>
                      <td>{it.status}</td>
                      <td>{it.number ?? ""}</td>
                      <td>
                        <button onClick={() => onOpenInvoice(it.id)}>Open</button>
                        {it.status === "draft" && (
                          <>
                            <button onClick={() => onIssueOne(it.id)}>Issue</button>
                            <button onClick={() => onDeleteDraft(it.id)}>Delete</button>
                          </>
                        )}
                        {it.status !== "draft" && (
                          <button onClick={() => onOpenPdf(it.id)}>PDF</button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
          )}

          {selectedInv && (
            <div className="panel">
              <h3>
                Invoice {selectedInv.number ? `#${selectedInv.number}` : "(draft)"} — {selectedInv.studentName} — {months[selectedInv.month - 1]} {selectedInv.year}
              </h3>

              <table>
                <thead>
                  <tr><th>Description</th><th style={{textAlign:"right"}}>Qty</th><th style={{textAlign:"right"}}>Unit</th><th style={{textAlign:"right"}}>Amount</th></tr>
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
                    <td colSpan={3} style={{ textAlign: "right" }}>Total:</td>
                    <td style={{ textAlign: "right" }}>{selectedInv.total.toFixed(2)}</td>
                  </tr>
                </tfoot>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
  
}
