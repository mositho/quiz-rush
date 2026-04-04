export interface Session {
  sessionId: string;
  status: "active" | "cooldown" | "finished";
  finishReason: string | null;
  startedAt: string;
  endsAt: string;
  cooldownUntil?: string | null;
  durationSeconds: number;
  selectedQuestionSetIds: readonly string[];
  currentQuestionIndex: number | null;
  totalQuestions: number;
  answeredQuestions: number;
  correctQuestions: number;
  wrongQuestions: number;
  currentScore: number;
  currentQuestion: Question | null;
}

export interface Question {
  position: number;
  questionId: string;
  questionSetId: string;
  difficulty: number;
  categories: readonly string[];
  text: string;
  options: readonly string[];
}

export interface SubmitAnswerResult {
  correct: boolean;
  awardedPoints: number;
  responseTimeMs: number;
  cooldownUntil: string | null;
  finished: boolean;
  finishReason: string | null;
}

export interface QuestionSet {
  id: string;
  name: string;
  description: string;
  length: number;
}

export interface PublicUser {
  publicUserId: string;
  displayName: string;
}

export interface ScoreSummary {
  scoreId: string;
  sessionId: string;
  finishedAt: string;
  finishReason: string;
  score: number;
  correctQuestions: number;
  wrongQuestions: number;
  answeredQuestions: number;
  totalQuestions: number;
  durationSeconds: number;
  playedMs: number;
  selectedQuestionSetIds: readonly string[];
  configurationKey: string;
}

export interface ScoreQuestionResult {
  questionId: string;
  questionSetId: string;
  correct: boolean;
  awardedPoints: number;
  responseTimeMs: number;
}

export interface ScoreDetail extends ScoreSummary {
  questionResults: ScoreQuestionResult[];
  player?: PublicUser;
}

export interface UserScoreList extends PublicUser {
  scores: ScoreSummary[];
}

export interface UserStats {
  gamesPlayed: number;
  bestScore: number;
  averageScore: number;
  totalCorrectQuestions: number;
}

export interface UserStatsProfile extends PublicUser {
  stats: UserStats;
}

export interface LeaderboardEntry {
  rank: number;
  scoreId: string;
  score: number;
  finishedAt: string;
  configurationKey: string;
  player: PublicUser;
}

export interface LeaderboardList {
  configurationKey?: string;
  entries: LeaderboardEntry[];
}

export interface LinkAccountResult {
  sessionId: string;
  scoreId: string;
  publicUserId: string;
  displayName: string;
  linked: boolean;
}
