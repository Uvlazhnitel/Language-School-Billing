import { useEffect, useMemo, useState } from "react";
import "./App.css";

import {
  fetchRows,
  saveCount,
  addOneMass,
  estimateBySchedule,
  setLocked,
  devSeed,
  devReset,
  deleteEnrollment,
  Row,
} from "./lib/attendance";

import {
  genDrafts,
  listInvoices,
  getInvoice,
  deleteDraft,
  issueOne,
  issueAll,
  InvoiceListItem,
  InvoiceDTO,
  InvoiceLine,
} from "./lib/invoice";

import {
  createPayment,
  invoiceSummary,
  listDebtors,
  listPayments,
  studentBalance,
  quickCash,
  DebtorDTO,
  PaymentDTO,
  BalanceDTO,
  InvoiceSummaryDTO,
} from "./lib/payments";

const months = [
  "January","February","March","April","May","June",
  "July","August","September","October","November","December"
];

type Tab = "attendance" | "invoices" | "debtors";

function todayYMD() {
  const d = new Date();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${mm}-${dd}`;
}

export default function App() {
  const now = new Date();
  const [tab, setTab] = useState<Tab>("attendance");
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth() + 1);

  // Attendance
  const [rows, setRows] = useState<Row[]>([]);
  const [loadingAtt, setLoadingAtt] = useState(false);
  const [courseFilter, setCourseFilter] = useState<number | undefined>(undefined);
  const [msg, setMsg] = useState("");

  // Invoices
  const [items, setItems] = useState<InvoiceListItem[]>([]);
  const [selectedInv, setSelectedInv] = useState<InvoiceDTO | null>(null);
  const [invSummary, setInvSummary] = useState<InvoiceSummaryDTO | null>(null);
  const [loadingInv, setLoadingInv] = useState(false);

  // Payment modal
  const [payOpen, setPayOpen] = useState(false);
  const [payAmount, setPayAmount] = useState(0);
  const [payMethod, setPayMethod] = useState<"cash" | "bank">("bank");
  const [payDate, setPayDate] = useState(todayYMD());
  const [payNote, setPayNote] = useState("");

  // Debtors
  const [debtors, setDebtors] = useState<DebtorDTO[]>([]);
  const [selectedStudentId, setSelectedStudentId] = useState<number | null>(null);
  const [selectedBalance, setSelectedBalance] = useState<BalanceDTO | null>(null);
  const [studentPayments, setStudentPayments] = useState<PaymentDTO[]>([]);
  const [quickAmount, setQuickAmount] = useState(0);
  const [quickNote, setQuickNote] = useState("Quick cash payment");

  // ---------------- Attendance ----------------
  async function loadAttendance() {
    setLoadingAtt(true);
    try {
      const data = await fetchRows(year, month, courseFilter);
      setRows(data);
    } finally {
      setLoadingAtt(false);
    }
  }

  useEffect(() => {
    if (tab === "attendance") loadAttendance();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, year, month, courseFilter]);

  const perLessonTotal = useMemo(
    () => rows.reduce((s, r) => s + r.count * r.lessonPrice, 0),
    [rows]
  );

  const onChangeCount = async (r: Row, v: number) => {
    const n = Number.isFinite(v) && v > 0 ? Math.trunc(v) : 0;
    await saveCount(r.studentId, r.courseId, year, month, n);
    setRows(rows.map(x => (x.enrollmentId === r.enrollmentId ? { ...x, count: n } : x)));
  };

  const onAddAll = async () => {
    await addOneMass(year, month, courseFilter);
    await loadAttendance();
  };

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
      if (r.count !== prev) {
        await saveCount(r.studentId, r.courseId, year, month, r.count);
      }
    }

    setMsg("Schedule hint applied");
    setTimeout(() => setMsg(""), 1500);
  };

  const onLock = async (lock: boolean) => {
    await setLocked(year, month, courseFilter, lock);
    await loadAttendance();
  };

  const onSeed = async () => {
    await devSeed();
    await loadAttendance();
  };

  const onReset = async () => {
    await devReset();
    await loadAttendance();
  };

  const onDeleteEnrollment = async (id: number) => {
    await deleteEnrollment(id);
    await loadAttendance();
  };

  // ---------------- Invoices ----------------
  async function loadInvoices() {
    setLoadingInv(true);
    try {
      // Show drafts for now. If you want issued/paid too, change to "all".
      const li = await listInvoices(year, month, "draft");
      setItems(li);
      setSelectedInv(null);
      setInvSummary(null);
    } finally {
      setLoadingInv(false);
    }
  }

  useEffect(() => {
    if (tab === "invoices") loadInvoices();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, year, month]);

  const onGenDrafts = async () => {
    const r = await genDrafts(year, month);
    // Your GenerateResult includes useful counters.
    alert(`Drafts generated:
created=${r.created}
updated=${r.updated}
skippedHasInvoice=${r.skippedHasInvoice}
skippedNoLines=${r.skippedNoLines}`);
    await loadInvoices();
  };

  const onOpenInvoice = async (id: number) => {
    const iv = await getInvoice(id);
    setSelectedInv(iv);
    setInvSummary(null);
  };

  const onDeleteDraft = async (id: number) => {
    await deleteDraft(id);
    await loadInvoices();
  };

  const onIssueOne = async (id: number) => {
    const r = await issueOne(id); // { number, pdfPath }
    alert(`Issued invoice ${r.number}\nPDF: ${r.pdfPath}`);
    await loadInvoices();
  };

  const onIssueAll = async () => {
    const r = await issueAll(year, month); // { count, pdfPaths }
    alert(`Issued: ${r.count}\nPDF:\n${r.pdfPaths.join("\n")}`);
    await loadInvoices();
  };

  // ---------------- Payments (invoice modal) ----------------
  async function openPayModalForInvoice(invoiceId: number) {
    const sum = await invoiceSummary(invoiceId);
    setInvSummary(sum);
    setPayAmount(sum.remaining);
    setPayMethod("bank");
    setPayDate(todayYMD());
    setPayNote(`Payment for invoice ${sum.number ?? invoiceId}`);
    setPayOpen(true);
  }

  async function submitPay() {
    if (!selectedInv) return;

    // We attach payment to invoice ID.
    const invId = selectedInv.id;
    const stId = selectedInv.studentId;

    await createPayment(stId, invId, payAmount, payMethod, payDate, payNote);

    const sum = await invoiceSummary(invId);
    setInvSummary(sum);

    if (sum.remaining <= 0.01) {
      alert("Invoice is fully paid.");
      setPayOpen(false);
    } else {
      alert(`Payment saved. Remaining: ${sum.remaining.toFixed(2)}`);
    }
  }

  // ---------------- Debtors ----------------
  async function loadDebtors() {
    const d = await listDebtors();
    setDebtors(d);
    setSelectedStudentId(null);
    setSelectedBalance(null);
    setStudentPayments([]);
  }

  useEffect(() => {
    if (tab === "debtors") loadDebtors();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab]);

  async function openStudent(studentId: number) {
    setSelectedStudentId(studentId);
    const b = await studentBalance(studentId);
    const ps = await listPayments(studentId);
    setSelectedBalance(b);
    setStudentPayments(ps);
    setQuickAmount(b.debt > 0 ? b.debt : 0);
  }

  async function submitQuickCash() {
    if (!selectedStudentId) return;
    await quickCash(selectedStudentId, quickAmount, quickNote);
    await openStudent(selectedStudentId);
    await loadDebtors();
  }

  // ---------------- Render ----------------
  return (
    <div className="container">
      <nav className="tabs">
        <button className={tab === "attendance" ? "active" : ""} onClick={() => setTab("attendance")}>
          Attendance
        </button>
        <button className={tab === "invoices" ? "active" : ""} onClick={() => setTab("invoices")}>
          Invoices
        </button>
        <button className={tab === "debtors" ? "active" : ""} onClick={() => setTab("debtors")}>
          Debtors
        </button>

        <div className="spacer" />

        <div className="monthpickers">
          <select value={month} onChange={(e) => setMonth(parseInt(e.target.value, 10))}>
            {months.map((m, i) => (
              <option key={m} value={i + 1}>{m}</option>
            ))}
          </select>
          <select value={year} onChange={(e) => setYear(parseInt(e.target.value, 10))}>
            {[year - 1, year, year + 1].map(y => (
              <option key={y} value={y}>{y}</option>
            ))}
          </select>
        </div>
      </nav>

      {/* Attendance */}
      {tab === "attendance" && (
        <>
          {msg && <div className="msg">{msg}</div>}

          <div className="controls">
            <button onClick={onSeed}>Seed demo</button>
            <button onClick={onReset}>Reset demo</button>
            <button onClick={onEstimate}>Schedule hint</button>
            <button onClick={onAddAll}>+1 all</button>
            <button onClick={() => onLock(true)}>Lock month</button>
            <button onClick={() => onLock(false)}>Unlock</button>
          </div>

          {loadingAtt ? (
            <div>Loading…</div>
          ) : rows.length === 0 ? (
            <div className="empty">No per-lesson rows. Create enrollments first.</div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Student</th>
                  <th>Course</th>
                  <th>Lesson price</th>
                  <th>Count</th>
                  <th>Total</th>
                  <th>Lock</th>
                  <th></th>
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
                        onChange={(e) => onChangeCount(r, parseInt(e.target.value, 10))}
                        style={{ width: "5rem", textAlign: "right" }}
                      />
                    </td>
                    <td style={{ textAlign: "right" }}>{(r.count * r.lessonPrice).toFixed(2)}</td>
                    <td>{r.locked ? "locked" : "open"}</td>
                    <td>
                      <button onClick={() => onDeleteEnrollment(r.enrollmentId)}>Delete</button>
                    </td>
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

      {/* Invoices */}
      {tab === "invoices" && (
        <>
          <div className="controls">
            <button onClick={onGenDrafts}>Build drafts</button>
            <button onClick={onIssueAll}>Issue all drafts</button>
            <button onClick={loadInvoices}>Refresh</button>
          </div>

          {loadingInv ? (
            <div>Loading…</div>
          ) : items.length === 0 ? (
            <div className="empty">No drafts. Click "Build drafts".</div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Student</th>
                  <th>Period</th>
                  <th>Lines</th>
                  <th>Total</th>
                  <th>Status</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {items.map(it => (
                  <tr key={it.id}>
                    <td>{it.studentName}</td>
                    <td>{months[it.month - 1]} {it.year}</td>
                    <td style={{ textAlign: "right" }}>{it.linesCount}</td>
                    <td style={{ textAlign: "right" }}>{it.total.toFixed(2)}</td>
                    <td>{it.status}</td>
                    <td>
                      <button onClick={() => onOpenInvoice(it.id)}>Open</button>
                      <button onClick={() => onIssueOne(it.id)}>Issue</button>
                      <button onClick={() => onDeleteDraft(it.id)}>Delete</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {selectedInv && (
            <div className="panel">
              <h3>
                Invoice (draft): {selectedInv.studentName} — {months[selectedInv.month - 1]} {selectedInv.year}
              </h3>

              <button
                onClick={() => openPayModalForInvoice(selectedInv.id)}
                style={{ marginBottom: 8 }}
              >
                Add payment (will issue invoice first)
              </button>

              <table>
                <thead>
                  <tr>
                    <th>Description</th>
                    <th>Qty</th>
                    <th>Unit</th>
                    <th>Amount</th>
                  </tr>
                </thead>
                <tbody>
                  {selectedInv.lines.map((l: InvoiceLine, idx: number) => (
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

              {invSummary && (
                <div style={{ marginTop: 10 }}>
                  <div><b>Payment summary</b></div>
                  <div>Total: {invSummary.total.toFixed(2)}</div>
                  <div>Paid: {invSummary.paid.toFixed(2)}</div>
                  <div>Remaining: {invSummary.remaining.toFixed(2)}</div>
                  <div>Status: {invSummary.status}</div>
                </div>
              )}
            </div>
          )}

          {payOpen && selectedInv && (
            <div className="modal">
              <div className="modalBody">
                <h3>Add payment</h3>

                <div className="formRow">
                  <label>Amount</label>
                  <input
                    type="number"
                    value={payAmount}
                    min={0}
                    step="0.01"
                    onChange={(e) => setPayAmount(parseFloat(e.target.value))}
                  />
                </div>

                <div className="formRow">
                  <label>Method</label>
                  <select value={payMethod} onChange={(e) => setPayMethod(e.target.value as "cash" | "bank")}>
                    <option value="bank">bank</option>
                    <option value="cash">cash</option>
                  </select>
                </div>

                <div className="formRow">
                  <label>Date</label>
                  <input type="date" value={payDate} onChange={(e) => setPayDate(e.target.value)} />
                </div>

                <div className="formRow">
                  <label>Note</label>
                  <input value={payNote} onChange={(e) => setPayNote(e.target.value)} />
                </div>

                <div className="modalActions">
                  <button
                    onClick={async () => {
                      // Ensure invoice is issued before attaching a payment.
                      await onIssueOne(selectedInv.id);
                      await submitPay();
                    }}
                  >
                    Save payment
                  </button>
                  <button onClick={() => setPayOpen(false)}>Close</button>
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {/* Debtors */}
      {tab === "debtors" && (
        <>
          <div className="controls">
            <button onClick={loadDebtors}>Refresh</button>
          </div>

          {debtors.length === 0 ? (
            <div className="empty">No debtors. Great!</div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Student</th>
                  <th style={{ textAlign: "right" }}>Debt</th>
                  <th style={{ textAlign: "right" }}>Invoiced</th>
                  <th style={{ textAlign: "right" }}>Paid</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {debtors.map(d => (
                  <tr key={d.studentId}>
                    <td>{d.studentName}</td>
                    <td style={{ textAlign: "right" }}>{d.debt.toFixed(2)}</td>
                    <td style={{ textAlign: "right" }}>{d.totalInvoiced.toFixed(2)}</td>
                    <td style={{ textAlign: "right" }}>{d.totalPaid.toFixed(2)}</td>
                    <td>
                      <button onClick={() => openStudent(d.studentId)}>Open</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {selectedStudentId && selectedBalance && (
            <div className="panel">
              <h3>Student: {selectedBalance.studentName}</h3>
              <div>Total invoiced: {selectedBalance.totalInvoiced.toFixed(2)}</div>
              <div>Total paid: {selectedBalance.totalPaid.toFixed(2)}</div>
              <div>Balance (paid - invoiced): {selectedBalance.balance.toFixed(2)}</div>
              <div><b>Debt:</b> {selectedBalance.debt.toFixed(2)}</div>

              <h4 style={{ marginTop: 10 }}>Quick cash</h4>
              <div className="formRow">
                <label>Amount</label>
                <input
                  type="number"
                  min={0}
                  step="0.01"
                  value={quickAmount}
                  onChange={(e) => setQuickAmount(parseFloat(e.target.value))}
                />
              </div>
              <div className="formRow">
                <label>Note</label>
                <input value={quickNote} onChange={(e) => setQuickNote(e.target.value)} />
              </div>
              <button onClick={submitQuickCash}>Save quick cash</button>

              <h4 style={{ marginTop: 14 }}>Payments</h4>
              {studentPayments.length === 0 ? (
                <div className="empty">No payments yet.</div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>Date</th>
                      <th>Method</th>
                      <th style={{ textAlign: "right" }}>Amount</th>
                      <th>Note</th>
                    </tr>
                  </thead>
                  <tbody>
                    {studentPayments.map(p => (
                      <tr key={p.id}>
                        <td>{p.paidAt.slice(0, 10)}</td>
                        <td>{p.method}</td>
                        <td style={{ textAlign: "right" }}>{p.amount.toFixed(2)}</td>
                        <td>{p.note}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
