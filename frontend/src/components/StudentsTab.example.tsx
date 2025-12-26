import { useState, useEffect } from "react";
import { listStudents, StudentDTO } from "../lib/students";

interface StudentsTabProps {
  showMessage: (text: string, type?: "success" | "error") => void;
  showConfirm: (message: string, onConfirm: () => void | Promise<void>, label?: string) => void;
}

/**
 * StudentsTab Component - EXAMPLE IMPLEMENTATION
 * 
 * This demonstrates how to extract tab functionality from the monolithic App.tsx.
 * Shows proper component structure, state management, and prop passing.
 * 
 * To use: Copy this pattern for other tabs (Courses, Enrollments, Attendance, Invoices)
 */
export function StudentsTab({ showMessage, showConfirm }: StudentsTabProps) {
  const [students, setStudents] = useState<StudentDTO[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [, setShowCreateForm] = useState(false);

  useEffect(() => {
    loadStudents();
  }, []);

  const loadStudents = async () => {
    try {
      const data = await listStudents(searchQuery, false);
      setStudents(data);
    } catch (error) {
      showMessage(`Failed to load students: ${error}`, "error");
    }
  };

  return (
    <div className="students-tab">
      <h2>Students (Example Component)</h2>
      <p style={{ background: "#ffe", padding: "10px", border: "1px solid #ee0" }}>
        This is an example component demonstrating the refactoring pattern.
        See FRONTEND_REFACTORING.md for complete implementation guide.
      </p>
      {/* Simplified UI for example purposes */}
      <div className="controls">
        <input
          type="text"
          placeholder="Search..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
        <button onClick={loadStudents}>Search</button>
        <button onClick={() => setShowCreateForm(true)}>+ New</button>
      </div>
      <div className="list">
        {students.map(s => (
          <div key={s.id}>
            {s.fullName} - {s.isActive ? "Active" : "Inactive"}
          </div>
        ))}
      </div>
    </div>
  );
}
