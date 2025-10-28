import { useEffect, useMemo, useState } from "react";
import "./App.css";
import { fetchRows, saveCount, addOneMass, estimateBySchedule, setLocked, seedDemo, Row } from "./lib/attendance";

const months = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];

function App() {
    const now = new Date();
    const [year, setYear] = useState(now.getFullYear());
    const [month, setMonth] = useState(now.getMonth()+1); // 1..12
    const [rows, setRows] = useState<Row[]>([]);
    const [loading, setLoading] = useState(false);
    const [courseFilter, setCourseFilter] = useState<number | undefined>(undefined);
    const [msg, setMsg] = useState("");

    async function load() {
        setLoading(true);
        try {
            const data = await fetchRows(year, month, courseFilter);
            setRows(data);
        } finally {
            setLoading(false);
        }
    }

    useEffect(() => { load(); }, [year, month, courseFilter]);

    const perLessonTotal = useMemo(() => rows.reduce((s, r) => s + r.count * r.lessonPrice, 0), [rows]);

    const onChangeCount = async (r: Row, v: number) => {
        const n = isNaN(v) || v < 0 ? 0 : Math.trunc(v);
        await saveCount(r.studentId, r.courseId, year, month, n);
        setRows(rows.map(x => (x.studentId===r.studentId && x.courseId===r.courseId ? { ...x, count: n } : x)));
    };

    const onAddAll = async () => {
        await addOneMass(year, month, courseFilter);
        await load();
    };

    const onEstimate = async () => {
        const hints = await estimateBySchedule(year, month, courseFilter);
        const patched = rows.map(r => {
            const key = `${r.studentId}-${r.courseId}`;
            const hint = hints[key] ?? 0;
            // substitute only if 0
            return r.count === 0 && hint > 0 ? { ...r, count: hint } : r;
        });
        setRows(patched);
        // save new values
        for (const r of patched) {
            if (r.count !== rows.find(x => x.studentId===r.studentId && x.courseId===r.courseId)?.count) {
                await saveCount(r.studentId, r.courseId, year, month, r.count);
            }
        }
        setMsg("Schedule hint applied");
        setTimeout(()=>setMsg(""), 1500);
    };

    const onLock = async (lock: boolean) => {
        await setLocked(year, month, courseFilter, lock);
        await load();
    };

    const onSeed = async () => {
        await seedDemo();
        await load();
    };

    return (
        <div className="container">
            <header>
                <h1>Attendance — {months[month-1]} {year}</h1>
                <div className="controls">
                    <select value={month} onChange={(e)=>setMonth(parseInt(e.target.value))}>
                        {months.map((m,i)=>(<option key={m} value={i+1}>{m}</option>))}
                    </select>
                    <select value={year} onChange={(e)=>setYear(parseInt(e.target.value))}>
                        {[year-1, year, year+1].map(y => (<option key={y} value={y}>{y}</option>))}
                    </select>
                    <button onClick={onSeed} title="Add demo data">Demo Data</button>
                    <button onClick={onEstimate}>Schedule Hint</button>
                    <button onClick={onAddAll}>+1 to All</button>
                    <button onClick={()=>onLock(true)}>Lock Month</button>
                    <button onClick={()=>onLock(false)}>Unlock</button>
                </div>
            </header>

            {msg && <div className="msg">{msg}</div>}

            {loading ? <div>Loading…</div> : (
                rows.length === 0 ? <div className="empty">No records (enrollments with "per lesson" payment are needed)</div> :
                <table>
                    <thead>
                        <tr>
                            <th>Student</th>
                            <th>Course</th>
                            <th>Lesson Price</th>
                            <th>Count</th>
                            <th>Total</th>
                            <th>Status</th>
                        </tr>
                    </thead>
                    <tbody>
                        {rows.map(r => (
                            <tr key={`${r.studentId}-${r.courseId}`}>
                                <td>{r.studentName}</td>
                                <td>{r.courseName} {r.courseType === "group" ? "(group)" : "(individual)"}</td>
                                <td style={{textAlign:"right"}}>{r.lessonPrice.toFixed(2)} €</td>
                                <td style={{textAlign:"right"}}>
                                    <input type="number" min={0} value={r.count}
                                                 disabled={r.locked}
                                                 onChange={(e)=>onChangeCount(r, parseInt(e.target.value))}
                                                 style={{width: "5rem", textAlign:"right"}} />
                                </td>
                                <td style={{textAlign:"right"}}>{(r.count * r.lessonPrice).toFixed(2)} €</td>
                                <td>{r.locked ? "🔒" : "✎"}</td>
                            </tr>
                        ))}
                    </tbody>
                    <tfoot>
                        <tr>
                            <td colSpan={4} style={{textAlign:"right"}}>Total per-lesson:</td>
                            <td style={{textAlign:"right"}}>{perLessonTotal.toFixed(2)} €</td>
                            <td></td>
                        </tr>
                    </tfoot>
                </table>
            )}
        </div>
    );
}

export default App;
