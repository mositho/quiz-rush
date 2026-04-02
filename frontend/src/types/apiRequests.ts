export interface StartSessionRequest {
  durationSeconds: number;
  selectedQuestionSetIds: string[];
}

export interface SubmitAnswerRequest {
  selectedAnswerIndex: number;
}
