import { useState, useEffect } from "react";
import { listStudents, createStudent, updateStudent, setStudentActive, deleteStudent, StudentDTO } from "../lib/students";

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
  const [editingStudent, setEditingStudent] = useState<StudentDTO | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [formFullName, setFormFullName] = useState("");
  const [formPhone, setFormPhone] = useState("");
  const [formEmail, setFormEmail] = useState("");
  const [formNote, setFormNote] = useState("");

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

  const handleCreate = async () => {
    try {
      await createStudent(formFullName, formPhone, formEmail, formNote);
      showMessage("Student created successfully", "success");
      setShowCreateForm(false);
      clearForm();
      loadStudents();
    } catch (error) {
      showMessage(`Failed to create student: ${error}`, "error");
    }
  };

  const clearForm = () => {
    setFormFullName("");
    setFormPhone("");
    setFormEmail("");
    setFormNote("");
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
