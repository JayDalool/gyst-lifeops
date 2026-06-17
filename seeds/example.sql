INSERT OR IGNORE INTO categories (name, type) VALUES
  ('personal', 'task'),
  ('health', 'goal'),
  ('coding', 'project'),
  ('sim rig', 'project'),
  ('business', 'general'),
  ('finance', 'goal'),
  ('other', 'general');

INSERT INTO tasks (title, description, category, priority, status, due_date, repeat_type, reminder_time) VALUES
  ('Morning reset checklist', 'Quick tidy, plan the day, clear the desk.', 'personal', 'medium', 'pending', date('now'), 'daily', '08:30'),
  ('Stretching session', 'Ten minutes of mobility work.', 'health', 'low', 'pending', date('now'), 'daily', '19:30'),
  ('Weekly planning review', 'Review projects, goals, and the next seven days.', 'personal', 'high', 'pending', date('now', 'weekday 1'), 'weekly', '18:00'),
  ('Monthly budget check-in', 'Review subscriptions, spending, and upcoming bills.', 'finance', 'medium', 'pending', date('now', 'start of month'), 'monthly', '17:00'),
  ('Renew insurance paperwork', 'Collect documents and confirm renewal details.', 'car', 'high', 'pending', date('now', '+4 days'), 'yearly', '09:00'),
  ('Overdue inbox cleanup', 'Clear important messages from the personal inbox.', 'personal', 'medium', 'pending', date('now', '-2 days'), 'none', '16:00');

INSERT INTO projects (name, description, category, status, start_date, target_date, notes) VALUES
  ('Sim Rig Build', 'Build out a clean sim racing setup from parts list through cable management.', 'sim rig', 'active', date('now', '-3 days'), date('now', '+45 days'), 'Keep the checklist practical: cockpit, wheelbase, pedals, display, cable routing, and calibration.'),
  ('LifeOps Tracker V1', 'Local-first productivity tracker with tasks, projects, goals, reminders, and charts.', 'coding', 'active', date('now', '-1 day'), date('now', '+14 days'), 'Version 1 stays web and SQLite only.');

INSERT INTO project_items (project_id, title, description, status, priority, due_date, sort_order)
SELECT id, 'Finalize parts list', 'Confirm wheelbase, pedals, mount, seat, and display plan.', 'completed', 'high', date('now', '-1 day'), 1
FROM projects WHERE name = 'Sim Rig Build';

INSERT INTO project_items (project_id, title, description, status, priority, due_date, sort_order)
SELECT id, 'Measure room layout', 'Check desk clearance, cable path, and monitor distance.', 'pending', 'medium', date('now', '+3 days'), 2
FROM projects WHERE name = 'Sim Rig Build';

INSERT INTO project_items (project_id, title, description, status, priority, due_date, sort_order)
SELECT id, 'Assemble cockpit frame', 'Build frame, level seat, tighten mounts.', 'pending', 'high', date('now', '+12 days'), 3
FROM projects WHERE name = 'Sim Rig Build';

INSERT INTO project_items (project_id, title, description, status, priority, due_date, sort_order)
SELECT id, 'Calibrate wheel and pedals', 'Set rotation, force feedback, brake pressure, and profiles.', 'pending', 'medium', date('now', '+20 days'), 4
FROM projects WHERE name = 'Sim Rig Build';

INSERT INTO project_items (project_id, title, description, status, priority, due_date, sort_order)
SELECT id, 'Create SQLite migrations', 'Schema for tasks, projects, goals, reminders, and sync metadata.', 'completed', 'high', date('now'), 1
FROM projects WHERE name = 'LifeOps Tracker V1';

INSERT INTO project_items (project_id, title, description, status, priority, due_date, sort_order)
SELECT id, 'Build task pages', 'Create, edit, complete, skip, and archive task flow.', 'pending', 'high', date('now', '+2 days'), 2
FROM projects WHERE name = 'LifeOps Tracker V1';

INSERT INTO project_items (project_id, title, description, status, priority, due_date, sort_order)
SELECT id, 'Polish dashboard charts', 'Check task, project, and goal graph rendering.', 'pending', 'medium', date('now', '+5 days'), 3
FROM projects WHERE name = 'LifeOps Tracker V1';

INSERT INTO goals (title, description, type, category, target_value, current_value, unit, start_date, end_date, status) VALUES
  ('Workout Sessions This Year', 'Track completed workout sessions as a consistency habit.', 'yearly', 'health', 120, 18, 'sessions', date('now', 'start of year'), date('now', 'start of year', '+1 year', '-1 day'), 'active'),
  ('Walking Days This Month', 'Count days with a purposeful walk.', 'monthly', 'health', 20, 6, 'days', date('now', 'start of month'), date('now', 'start of month', '+1 month', '-1 day'), 'active'),
  ('Coding Focus Hours', 'Deep work hours for shipping personal projects.', 'monthly', 'coding', 60, 14, 'hours', date('now', 'start of month'), date('now', 'start of month', '+1 month', '-1 day'), 'active');

INSERT INTO goal_progress_entries (goal_id, entry_date, value, notes)
SELECT id, date('now', 'start of year'), 4, 'January baseline'
FROM goals WHERE title = 'Workout Sessions This Year';

INSERT INTO goal_progress_entries (goal_id, entry_date, value, notes)
SELECT id, date('now', 'start of year', '+2 months'), 10, 'Quarter ramp'
FROM goals WHERE title = 'Workout Sessions This Year';

INSERT INTO goal_progress_entries (goal_id, entry_date, value, notes)
SELECT id, date('now'), 18, 'Current check-in'
FROM goals WHERE title = 'Workout Sessions This Year';

INSERT INTO goal_progress_entries (goal_id, entry_date, value, notes)
SELECT id, date('now', 'start of month'), 2, 'Month start'
FROM goals WHERE title = 'Walking Days This Month';

INSERT INTO goal_progress_entries (goal_id, entry_date, value, notes)
SELECT id, date('now'), 6, 'Current check-in'
FROM goals WHERE title = 'Walking Days This Month';

INSERT INTO reminders (source_type, source_id, title, reminder_date, reminder_time, status)
SELECT 'task', id, title, due_date, reminder_time, 'active'
FROM tasks
WHERE reminder_time IS NOT NULL;

INSERT INTO reminders (source_type, source_id, title, reminder_date, reminder_time, status) VALUES
  ('manual', NULL, 'Review yearly goals', date('now'), '20:00', 'active');
