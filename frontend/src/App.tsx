import { useEffect, useMemo, useState } from "react";
import "./App.css";
import {
    fetchRows, saveCount, addOneMass, estimateBySchedule, setLocked,
    devSeed, devReset, deleteEnrollment, Row
} from "./lib/attendance";
import { genDrafts, listDrafts, getInvoice, deleteDraft, InvoiceListItem, InvoiceDTO, InvoiceLine } from "./lib/invoice";

const months = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];

type Tab = "attendance" | "invoices";

function App() {
    const now = new Date();
    const [tab, setTab] = useState<Tab>("attendance");
    const [year, setYear] = useState(now.getFullYear());
    const [month, setMonth] = useState(now.getMonth() + 1); // 1..12

    // Attendance state
    const [rows, setRows] = useState<Row[]>([]);
    const [loading, setLoading] = useState(false);
    const [courseFilter, setCourseFilter] = useState<number | undefined>(undefined);
    const [msg, setMsg] = useState("");

    // Invoices state
    const [items, setItems] = useState<InvoiceListItem[]>([]);
    const [selected, setSelected] = useState<InvoiceDTO | null>(null);
    const [loadingInv, setLoadingInv] = useState(false);

    // --- Attendance loaders ---
    async function loadAttendance() {
        setLoading(true);
        try {
            const data = await fetchRows(year, month, courseFilter);
            setRows(data);
        } finally {
            setLoading(false);
        }
    }
    useEffect(() => { if (tab === "attendance") loadAttendance(); }, [tab, year, month, courseFilter]);

    const perLessonTotal = useMemo(() => rows.reduce((s, r) => s + r.count * r.lessonPrice, 0), [rows]);

    const onChangeCount = async (r: Row, v: number) => {
        const n = isNaN(v) || v < 0 ? 0 : Math.trunc(v);
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
    const onSeed = async () => { await devSeed(); await loadAttendance(); };
    const onReset = async () => { await devReset(); await loadAttendance(); };
    const onDeleteEnrollment = async (id: number) => { await deleteEnrollment(id); await loadAttendance(); };

    // --- Invoices loaders ---
    async function loadInvoices() {
        setLoadingInv(true);
        try {
            const li = await listDrafts(year, month);
            setItems(li);
            setSelected(null);
        } finally {
            setLoadingInv(false);
        }
    }
    useEffect(() => { if (tab === "invoices") loadInvoices(); }, [tab, year, month]);

    const onGenDrafts = async () => {
        await genDrafts(year, month);
        await loadInvoices();
    };
    const onOpenInvoice = async (id: number) => {
        const iv = await getInvoice(id);
        setSelected(iv);
    };
    const onDeleteDraft = async (id: number) => {
        await deleteDraft(id);
        await loadInvoices();
    };

    return (
        <div className="container">
            <nav className="tabs">
                <button className={tab === "attendance" ? "active" : ""} onClick={() => setTab("attendance")}>Attendance</button>
                <button className={tab === "invoices" ? "active" : ""} onClick={() => setTab("invoices")}>Invoices</button>
                <div className="spacer" />
                <div className="monthpickers">
                    <select value={month} onChange={(e) => setMonth(parseInt(e.target.value))}>
                        {months.map((m, i) => (<option key={m} value={i + 1}>{m}</option>))}
                    </select>
                    <select value={year} onChange={(e) => setYear(parseInt(e.target.value))}>
                        {[year - 1, year, year + 1].map(y => (<option key={y} value={y}>{y}</option>))}
                    </select>
                </div>
            </nav>

            {tab === "attendance" && (
                <>
                    {msg && <div className="msg">{msg}</div>}
                    <div className="controls">
                        <button onClick={onSeed}>Demo data</button>
                        <button onClick={onReset}>Reset demo</button>
                        <button onClick={onEstimate}>Schedule hint</button>
                        <button onClick={onAddAll}>+1 to all</button>
                        <button onClick={() => onLock(true)}>Lock month</button>
                        <button onClick={() => onLock(false)}>Unlock</button>
                    </div>
                    {loading ? <div>Loading…</div> : (
                        rows.length === 0 ? <div className="empty">No records (enrollments with "per lesson" payment are needed)</div> :
                        <table>
                            <thead>
                                <tr>
                                    <th>Student</th><th>Course</th><th>Lesson price</th><th>Count</th><th>Total</th><th>Status</th><th></th>
                                </tr>
                            </thead>
                            <tbody>
                                {rows.map(r => (
                                    <tr key={r.enrollmentId}>
                                        <td>{r.studentName}</td>
                                        <td>{r.courseName} {r.courseType === "group" ? "(group)" : "(individual)"}</td>
                                        <td style={{ textAlign: "right" }}>{r.lessonPrice.toFixed(2)} €</td>
                                        <td style={{ textAlign: "right" }}>
                                            <input type="number" min={0} value={r.count}
                                                disabled={r.locked}
                                                onChange={(e) => onChangeCount(r, parseInt(e.target.value))}
                                                style={{ width: "5rem", textAlign: "right" }} />
                                        </td>
                                        <td style={{ textAlign: "right" }}>{(r.count * r.lessonPrice).toFixed(2)} €</td>
                                        <td>{r.locked ? "🔒" : "✎"}</td>
                                        <td><button onClick={() => onDeleteEnrollment(r.enrollmentId)}>Delete</button></td>
                                    </tr>
                                ))}
                            </tbody>
                            <tfoot>
                                <tr>
                                    <td colSpan={4} style={{ textAlign: "right" }}>Total per-lesson:</td>
                                    <td style={{ textAlign: "right" }}>{perLessonTotal.toFixed(2)} €</td>
                                    <td colSpan={2}></td>
                                </tr>
                            </tfoot>
                        </table>
                    )}
                </>
            )}

            {tab === "invoices" && (
                <>
                    <div className="controls">
                        <button onClick={onGenDrafts}>Generate drafts</button>
                        <button onClick={loadInvoices}>Refresh</button>
                    </div>

                    {loadingInv ? <div>Loading…</div> : (
                        items.length === 0 ? <div className="empty">No drafts. Click "Generate drafts".</div> :
                        <table>
                            <thead>
                                <tr>
                                    <th>Student</th><th>Month</th><th>Lines</th><th>Total</th><th>Status</th><th></th>
                                </tr>
                            </thead>
                            <tbody>
                                {items.map(it => (
                                    <tr key={it.id}>
                                        <td>{it.studentName}</td>
                                        <td>{months[it.month - 1]} {it.year}</td>
                                        <td style={{ textAlign: "right" }}>{it.linesCount}</td>
                                        <td style={{ textAlign: "right" }}>{it.total.toFixed(2)} €</td>
                                        <td>{it.status}</td>
                                        <td>
                                            <button onClick={() => onOpenInvoice(it.id)}>Open</button>
                                            <button onClick={() => onDeleteDraft(it.id)}>Delete</button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    )}

                    {selected && (
                        <div className="panel">
                            <h3>Invoice (draft): {selected.studentName} — {months[selected.month - 1]} {selected.year}</h3>
                            <table>
                                <thead>
                                    <tr><th>Description</th><th>Quantity</th><th>Price</th><th>Total</th></tr>
                                </thead>
                                <tbody>
                                    {selected.lines.map((l: InvoiceLine, idx: number) => (
                                        <tr key={idx}>
                                            <td>{l.description}</td>
                                            <td style={{ textAlign: "right" }}>{l.qty}</td>
                                            <td style={{ textAlign: "right" }}>{l.unitPrice.toFixed(2)} €</td>
                                            <td style={{ textAlign: "right" }}>{l.amount.toFixed(2)} €</td>
                                        </tr>
                                    ))}
                                </tbody>
                                <tfoot>
                                    <tr>
                                        <td colSpan={3} style={{ textAlign: "right" }}>Total:</td>
                                        <td style={{ textAlign: "right" }}>{selected.total.toFixed(2)} €</td>
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

export default App;
