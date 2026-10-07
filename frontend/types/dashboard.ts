export interface DashboardCounts {
  active_students: number;
  active_staff: number;
  active_teachers: number;
  total_classes: number;
  active_enrollments: number;
}

export interface AcademicContext {
  academic_year_id?: string;
  academic_year_name?: string;
  term_id?: string;
  term_name?: string;
}

export interface AttendanceSummary {
  date: string;
  total_sessions: number;
  present_count: number;
  absent_count: number;
  late_count: number;
  excused_count: number;
  total_marks: number;
  attendance_rate: number;
}

export interface RecentAnnouncement {
  id: string;
  title: string;
  status: string;
  target_type: string;
  target_role?: string;
  target_class_name?: string;
  priority: string;
  published_at?: string;
  created_at: string;
  author_name: string;
}

export interface RecentActivity {
  id: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  actor_email: string;
  actor_name: string;
  created_at: string;
  metadata?: Record<string, unknown>;
}

export interface QuickAction {
  id: string;
  title: string;
  description: string;
  href: string;
  category: string;
}

export interface AdminDashboardData {
  counts: DashboardCounts;
  academic_context: AcademicContext;
  attendance: AttendanceSummary;
  recent_announcements: RecentAnnouncement[];
  recent_activity: RecentActivity[];
  quick_actions: QuickAction[];
}
