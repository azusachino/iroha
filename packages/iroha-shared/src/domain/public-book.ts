// The public reading list is a privacy-trimmed projection of the canonical
// media records: public identity and reading dates only. Keeping its wire
// shape here prevents the public site from inventing a second frontend-only
// contract. Source ids, notes, ratings, reading time and progress never ship.
export interface PublicBook {
  id: string;
  title: string;
  authors: string[];
  status:
    | "planning"
    | "in_progress"
    | "completed"
    | "paused"
    | "abandoned"
    | "dropped";
  started_at?: string;
  finished_at?: string;
  last_read_at?: string;
}

export type PublicBookStatus = PublicBook["status"];

export const PUBLIC_BOOK_STATUS_LABELS: Record<PublicBookStatus, string> = {
  planning: "Planned",
  in_progress: "Reading",
  completed: "Finished",
  paused: "Paused",
  abandoned: "Abandoned",
  dropped: "Dropped",
};
