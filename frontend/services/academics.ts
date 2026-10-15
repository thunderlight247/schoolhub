import { apiGet, apiPost, apiPatch, apiPut, apiDelete } from "@/lib/api";
import type {
  AcademicYear,
  Class,
  ClassSubject,
  Programme,
  Subject,
  Term,
} from "@/types/academics";

// Academic endpoints wrap lists in plural keys and single results in
// singular keys ({"academic_years": [...]}, {"academic_year": {...}}, …).

interface YearEnvelope {
  academic_year: AcademicYear;
}
interface TermEnvelope {
  term: Term;
}
interface ProgrammeEnvelope {
  programme: Programme;
}
interface SubjectEnvelope {
  subject: Subject;
}
interface ClassEnvelope {
  class: Class;
}

// --- academic years ---

export function listAcademicYears(): Promise<AcademicYear[]> {
  return apiGet<{ academic_years: AcademicYear[] }>("/academic-years").then(
    (r) => r.academic_years,
  );
}

export function createAcademicYear(
  name: string,
  start_date: string,
  end_date: string,
  is_current: boolean,
): Promise<AcademicYear> {
  return apiPost<YearEnvelope>("/academic-years", {
    name,
    start_date,
    end_date,
    is_current,
  }).then((r) => r.academic_year);
}

export function updateAcademicYear(
  id: string,
  body: { name?: string; start_date?: string; end_date?: string },
): Promise<AcademicYear> {
  return apiPatch<YearEnvelope>(`/academic-years/${id}`, body).then((r) => r.academic_year);
}

export function setCurrentAcademicYear(id: string): Promise<AcademicYear> {
  return apiPut<YearEnvelope>(`/academic-years/${id}/current`).then((r) => r.academic_year);
}

// --- terms ---

export function listTerms(): Promise<Term[]> {
  return apiGet<{ terms: Term[] }>("/terms").then((r) => r.terms);
}

export function createTerm(
  academic_year_id: string,
  name: string,
  start_date: string,
  end_date: string,
  is_current: boolean,
): Promise<Term> {
  return apiPost<TermEnvelope>("/terms", {
    academic_year_id,
    name,
    start_date,
    end_date,
    is_current,
  }).then((r) => r.term);
}

export function updateTerm(
  id: string,
  body: { name?: string; start_date?: string; end_date?: string },
): Promise<Term> {
  return apiPatch<TermEnvelope>(`/terms/${id}`, body).then((r) => r.term);
}

export function setCurrentTerm(id: string): Promise<Term> {
  return apiPut<TermEnvelope>(`/terms/${id}/current`).then((r) => r.term);
}

// --- programmes ---

export function listProgrammes(): Promise<Programme[]> {
  return apiGet<{ programmes: Programme[] }>("/programmes").then((r) => r.programmes);
}

export function createProgramme(code: string, name: string, description?: string): Promise<Programme> {
  return apiPost<ProgrammeEnvelope>("/programmes", { code, name, description }).then(
    (r) => r.programme,
  );
}

export function updateProgramme(
  id: string,
  body: { code?: string; name?: string; description?: string | null },
): Promise<Programme> {
  return apiPatch<ProgrammeEnvelope>(`/programmes/${id}`, body).then((r) => r.programme);
}

// --- subjects ---

export function listSubjects(): Promise<Subject[]> {
  return apiGet<{ subjects: Subject[] }>("/subjects").then((r) => r.subjects);
}

export function createSubject(
  code: string,
  name: string,
  type: string,
  description?: string,
  department_id?: string,
): Promise<Subject> {
  return apiPost<SubjectEnvelope>("/subjects", {
    code,
    name,
    type,
    description,
    department_id,
  }).then((r) => r.subject);
}

export function updateSubject(
  id: string,
  body: { code?: string; name?: string; description?: string | null; type?: string },
): Promise<Subject> {
  return apiPatch<SubjectEnvelope>(`/subjects/${id}`, body).then((r) => r.subject);
}

// --- classes ---

export function listClasses(): Promise<Class[]> {
  return apiGet<{ classes: Class[] }>("/classes").then((r) => r.classes);
}

export function createClass(body: {
  academic_year_id: string;
  programme_id?: string;
  name: string;
  level?: string;
  capacity?: number;
}): Promise<Class> {
  return apiPost<ClassEnvelope>("/classes", body).then((r) => r.class);
}

export function updateClass(
  id: string,
  body: { programme_id?: string | null; name?: string; level?: string | null; capacity?: number | null },
): Promise<Class> {
  return apiPatch<ClassEnvelope>(`/classes/${id}`, body).then((r) => r.class);
}

// --- class curriculum (class_subjects) ---

export function listClassSubjects(classId: string): Promise<ClassSubject[]> {
  return apiGet<{ subjects: ClassSubject[] }>(`/classes/${classId}/subjects`).then(
    (r) => r.subjects,
  );
}

export function assignSubjectToClass(classId: string, subjectId: string): Promise<void> {
  return apiPost(`/classes/${classId}/subjects`, { subject_id: subjectId });
}

export function removeSubjectFromClass(classId: string, subjectId: string): Promise<void> {
  return apiDelete(`/classes/${classId}/subjects/${subjectId}`);
}
