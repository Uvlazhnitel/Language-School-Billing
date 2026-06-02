package main

import (
	"context"

	"langschool/ent"
	appruntime "langschool/internal/runtime"
)

// migrateLegacyCourseTeachers converts legacy course.teacher_name values into
// Teacher records and links courses through teacher_id. It is idempotent.
func migrateLegacyCourseTeachers(ctx context.Context, client *ent.Client) error {
	return appruntime.MigrateLegacyCourseTeachers(ctx, client)
}
