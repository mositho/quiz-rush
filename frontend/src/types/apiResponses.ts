export interface Session {
  sessionId: string;
  status: "active" | "cooldown" | "finished";
  finishReason: string | null;
  startedAt: string;
  endsAt: string;
  cooldownUntil: string | null;
  durationSeconds: number;
  selectedQuestionSetIds: string[];
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
  nextQuestion: Question | null;
  finished: boolean;
  finishReason: string | null;
}

export interface QuestionSet {
  id: string;
  name: string;
  description: string;
  length: number;
}
