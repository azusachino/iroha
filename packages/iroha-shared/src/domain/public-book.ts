// The public reading list is a privacy-trimmed projection of the canonical
// media records: public identity and reading dates only. Keeping its wire
// shape here prevents the public site from inventing a second frontend-only
// contract. Source ids, notes, ratings, reading time and progress never ship.
// Only completed books are public; the server decides that, not the page.
export interface PublicBook {
  id: string;
  title: string;
  authors: string[];
  status: "completed";
  started_at?: string;
  finished_at?: string;
}
