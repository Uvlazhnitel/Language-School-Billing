package main

import (
	"context"

	"langschool/ent"
	"langschool/ent/course"
	"langschool/ent/teacher"
)

// migrateLegacyCourseTeachers converts legacy course.teacher_name values into
// Teacher records and links courses through teacher_id. It is idempotent.
func migrateLegacyCourseTeachers(ctx context.Context, client *ent.Client) error {
	legacyCourses, err := client.Course.Query().
		Where(
			course.TeacherIDIsNil(),
			course.TeacherNameNEQ(""),
		).
		All(ctx)
	if err != nil {
		return err
	}

	for _, c := range legacyCourses {
		tch, err := client.Teacher.Query().
			Where(teacher.FullNameEqualFold(c.TeacherName)).
			Only(ctx)
		if err != nil {
			if !ent.IsNotFound(err) {
				return err
			}
			tch, err = client.Teacher.Create().
				SetFullName(c.TeacherName).
				SetIsActive(true).
				Save(ctx)
			if err != nil {
				return err
			}
		}

		if _, err := client.Course.UpdateOneID(c.ID).
			SetTeacherID(tch.ID).
			Save(ctx); err != nil {
			return err
		}
	}

	return nil
}
