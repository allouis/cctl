/**
 * Fuzzy matching and recency logic for repository selection.
 */

const RECENT_REPOS_KEY = "cctl:recent-repos";
const MAX_RECENTS = 5;

export interface ScoredRepo {
  path: string;
  basename: string;
  score: number;
}

/**
 * Score a repo path against a query using fuzzy matching.
 * Higher score = better match. Returns 0 for no match.
 */
export function fuzzyScore(path: string, query: string): number {
  if (!query) return 1;

  const lowerPath = path.toLowerCase();
  const lowerBasename = path.split("/").pop()!.toLowerCase();
  const lowerQuery = query.toLowerCase();

  // Exact basename match is best
  if (lowerBasename === lowerQuery) return 100;

  // Basename starts with query
  if (lowerBasename.startsWith(lowerQuery)) return 80;

  // Basename contains query
  if (lowerBasename.includes(lowerQuery)) return 60;

  // Full path contains query
  if (lowerPath.includes(lowerQuery)) return 40;

  // Character-by-character fuzzy match on basename
  let qi = 0;
  for (let i = 0; i < lowerBasename.length && qi < lowerQuery.length; i++) {
    if (lowerBasename[i] === lowerQuery[qi]) qi++;
  }
  if (qi === lowerQuery.length) return 20;

  // Same against full path
  qi = 0;
  for (let i = 0; i < lowerPath.length && qi < lowerQuery.length; i++) {
    if (lowerPath[i] === lowerQuery[qi]) qi++;
  }
  if (qi === lowerQuery.length) return 10;

  return 0;
}

export function searchRepos(repos: string[], query: string, limit: number): ScoredRepo[] {
  return repos
    .map((path) => ({
      path,
      basename: path.split("/").pop()!,
      score: fuzzyScore(path, query),
    }))
    .filter((r) => r.score > 0)
    .sort((a, b) => b.score - a.score)
    .slice(0, limit);
}

export function getRecentRepoPaths(): string[] {
  try {
    const raw = localStorage.getItem(RECENT_REPOS_KEY);
    return raw ? (JSON.parse(raw) as string[]) : [];
  } catch {
    return [];
  }
}

export function addRecentRepo(path: string): void {
  const recents = getRecentRepoPaths().filter((r) => r !== path);
  recents.unshift(path);
  if (recents.length > MAX_RECENTS) recents.length = MAX_RECENTS;
  localStorage.setItem(RECENT_REPOS_KEY, JSON.stringify(recents));
}

/**
 * Return recent repos that still exist in the available repos list.
 */
export function getRecentRepos(repos: string[]): string[] {
  const set = new Set(repos);
  return getRecentRepoPaths().filter((r) => set.has(r));
}
