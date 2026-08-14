// The public site uses the same canonical display rules as the private
// cockpit. Keep this adapter so existing public components do not know the
// package import path.
export {
  formatDate,
  formatDateOnly,
  formatDistance,
  formatDuration,
  formatElevation,
  formatHr,
  formatPace,
  formatSport,
  formatSwimmingPace,
} from "@iroha/shared/format";
