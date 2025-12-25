# Frontend Refactoring Guide: Splitting App.tsx

## Current State

**Problem:** `App.tsx` is 1,217 lines with 5 distinct tabs all in one component.

**Structure:**
- Students tab: ~87 lines (lines 726-813)
- Courses tab: ~96 lines (lines 813-909)
- Enrollments tab: ~126 lines (lines 909-1035)
- Attendance tab: ~81 lines (lines 1035-1116)
- Invoice tab: ~100 lines (lines 1116-1217)

## Proposed Solution

Split into separate component files:

```
frontend/src/
├── App.tsx (main orchestrator, ~150 lines)
├── components/
│   ├── StudentsTab.tsx
│   ├── CoursesTab.tsx
│   ├── EnrollmentsTab.tsx
│   ├── AttendanceTab.tsx
│   └── InvoicesTab.tsx
├── hooks/
│   ├── useMessage.ts (message state management)
│   └── useConfirmDialog.ts (confirmation dialog state)
└── types/
    └── common.ts (shared types)
```

## Step-by-Step Refactoring Plan

### Phase 1: Extract Shared Utilities (Low Risk)

1. **Create `hooks/useMessage.ts`**
   ```typescript
   export function useMessage() {
     const [message, setMessage] = useState<{
       text: string;
       type: "success" | "error";
     } | null>(null);
     
     const messageTimeoutRef = useRef<number | null>(null);
     
     const showMessage = useCallback((text: string, type: "success" | "error" = "success") => {
       // ... (extract from App.tsx)
     }, []);
     
     return { message, showMessage };
   }
   ```

2. **Create `hooks/useConfirmDialog.ts`**
   ```typescript
   export function useConfirmDialog() {
     const [confirmDialog, setConfirmDialog] = useState<{
       isOpen: boolean;
       message: string;
       onConfirm: () => void | Promise<void>;
       confirmButtonLabel?: string;
     } | null>(null);
     
     const showConfirm = (messageText: string, onConfirm: () => void | Promise<void>, confirmButtonLabel?: string) => {
       setConfirmDialog({ isOpen: true, message: messageText, onConfirm, confirmButtonLabel });
     };
     
     // ... handle functions
     
     return { confirmDialog, showConfirm, handleConfirmYes, handleConfirmNo };
   }
   ```

### Phase 2: Extract Tab Components (Medium Risk)

3. **Create `components/StudentsTab.tsx`**
   ```typescript
   interface StudentsTabProps {
     showMessage: (text: string, type?: "success" | "error") => void;
     showConfirm: (message: string, onConfirm: () => void | Promise<void>, label?: string) => void;
   }
   
   export function StudentsTab({ showMessage, showConfirm }: StudentsTabProps) {
     // Extract all students-related state
     const [students, setStudents] = useState<StudentDTO[]>([]);
     const [searchQuery, setSearchQuery] = useState("");
     // ... etc
     
     // Extract all students-related functions
     const loadStudents = async () => { /* ... */ };
     const handleCreate = async () => { /* ... */ };
     // ... etc
     
     return (
       <div>
         {/* Extract JSX from lines 726-813 */}
       </div>
     );
   }
   ```

4. **Repeat for other tabs**
   - `CoursesTab.tsx` (lines 813-909)
   - `EnrollmentsTab.tsx` (lines 909-1035)
   - `AttendanceTab.tsx` (lines 1035-1116)
   - `InvoicesTab.tsx` (lines 1116-1217)

### Phase 3: Simplify Main App.tsx (Low Risk)

5. **Update `App.tsx`**
   ```typescript
   import { StudentsTab } from "./components/StudentsTab";
   import { CoursesTab } from "./components/CoursesTab";
   import { EnrollmentsTab } from "./components/EnrollmentsTab";
   import { AttendanceTab } from "./components/AttendanceTab";
   import { InvoicesTab } from "./components/InvoicesTab";
   import { useMessage } from "./hooks/useMessage";
   import { useConfirmDialog } from "./hooks/useConfirmDialog";
   
   export default function App() {
     const [tab, setTab] = useState<Tab>("students");
     const { message, showMessage } = useMessage();
     const { confirmDialog, showConfirm, handleConfirmYes, handleConfirmNo } = useConfirmDialog();
     
     return (
       <div className="app">
         {/* Message display */}
         {message && <div className={`message ${message.type}`}>{message.text}</div>}
         
         {/* Confirmation dialog */}
         {confirmDialog?.isOpen && <div className="confirm-dialog">...</div>}
         
         {/* Tab navigation */}
         <div className="tabs">
           <button className={tab === "students" ? "active" : ""} onClick={() => setTab("students")}>
             Students
           </button>
           {/* ... other tabs */}
         </div>
         
         {/* Tab content */}
         {tab === "students" && <StudentsTab showMessage={showMessage} showConfirm={showConfirm} />}
         {tab === "courses" && <CoursesTab showMessage={showMessage} showConfirm={showConfirm} />}
         {tab === "enrollments" && <EnrollmentsTab showMessage={showMessage} showConfirm={showConfirm} />}
         {tab === "attendance" && <AttendanceTab showMessage={showMessage} showConfirm={showConfirm} />}
         {tab === "invoice" && <InvoicesTab showMessage={showMessage} showConfirm={showConfirm} />}
       </div>
     );
   }
   ```

## Benefits

After refactoring:
- **Maintainability**: Each tab is self-contained and easier to understand
- **Testability**: Individual components can be tested in isolation
- **Performance**: Unused tab components aren't re-rendered
- **Collaboration**: Multiple developers can work on different tabs simultaneously
- **Code reuse**: Shared hooks can be reused across components

## Estimated Effort

- Phase 1 (Hooks extraction): 1-2 hours
- Phase 2 (Tab components): 3-4 hours
- Phase 3 (Main App update): 1 hour
- Testing: 2-3 hours
- **Total**: 7-10 hours

## Testing Strategy

After each phase:
1. Run `npm run build` to check for TypeScript errors
2. Test the modified tab functionality manually
3. Ensure no regressions in other tabs
4. Check that state management still works correctly

## Example: Students Tab Component

See `components/StudentsTab.example.tsx` for a complete example implementation.

## Migration Checklist

- [ ] Create `frontend/src/hooks/` directory
- [ ] Create `frontend/src/components/` directory
- [ ] Create `frontend/src/types/` directory
- [ ] Extract `useMessage` hook
- [ ] Extract `useConfirmDialog` hook
- [ ] Create `StudentsTab` component
- [ ] Create `CoursesTab` component
- [ ] Create `EnrollmentsTab` component
- [ ] Create `AttendanceTab` component
- [ ] Create `InvoicesTab` component
- [ ] Update main `App.tsx` to use new components
- [ ] Test each tab individually
- [ ] Run full application test
- [ ] Update CI/CD to include new files

## Notes

- Keep this as a gradual refactoring - do one tab at a time
- Each commit should be a working state
- Use feature flags if deploying during refactoring
- Consider using React Context for shared state if needed
- Can further split into smaller components (e.g., StudentForm, StudentList)

## Additional Improvements (Optional)

After basic split:
- Add React Query for data fetching
- Add form validation library (e.g., Zod, Yup)
- Extract common UI components (Button, Input, Modal)
- Add loading states and error boundaries
- Implement optimistic updates
