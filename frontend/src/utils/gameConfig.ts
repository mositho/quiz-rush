import type { QuestionSet, ScoreSummary } from "@/types/apiResponses";

export const TIMER_PRESETS = [60, 120, 180, 300] as const;
export const DEFAULT_DURATION_SECONDS = 180;
export const LEADERBOARD_LIMIT = 25;

export function buildConfigurationKey(
  durationSeconds: number,
  selectedQuestionSetIds: readonly string[]
) {
  const normalizedSetIds = [...selectedQuestionSetIds].sort();
  return `duration=${durationSeconds}|sets=${normalizedSetIds.join(",")}`;
}

export function parseConfigurationKey(configurationKey: string) {
  const [durationPart = "", setPart = ""] = configurationKey.split("|");
  const durationSeconds = Number(durationPart.replace("duration=", "")) || DEFAULT_DURATION_SECONDS;
  const selectedQuestionSetIds = setPart
    .replace("sets=", "")
    .split(",")
    .map((entry) => entry.trim())
    .filter(Boolean);

  return { durationSeconds, selectedQuestionSetIds };
}

export function formatDurationLabel(durationSeconds: number) {
  const minutes = Math.floor(durationSeconds / 60);
  const seconds = durationSeconds % 60;

  if (seconds === 0) {
    return `${minutes} min`;
  }

  return `${minutes}:${String(seconds).padStart(2, "0")}`;
}

export function formatPlayedMs(playedMs: number) {
  const totalSeconds = Math.max(0, Math.round(playedMs / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  return `${minutes}:${String(seconds).padStart(2, "0")}`;
}

export function describeQuestionSets(
  selectedQuestionSetIds: readonly string[],
  questionSets: QuestionSet[]
) {
  if (selectedQuestionSetIds.length === 0) {
    return "No sets";
  }

  const names = selectedQuestionSetIds.map((setId) => {
    return questionSets.find((entry) => entry.id === setId)?.name ?? setId;
  });

  return names.join(", ");
}

export function findBestScoreForConfig(scores: ScoreSummary[], configurationKey: string) {
  return [...scores]
    .filter((score) => score.configurationKey === configurationKey)
    .sort(
      (left, right) => right.score - left.score || left.finishedAt.localeCompare(right.finishedAt)
    )[0];
}
