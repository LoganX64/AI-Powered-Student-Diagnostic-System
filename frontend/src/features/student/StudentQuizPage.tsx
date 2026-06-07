import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../../components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "../../components/ui/card";
import { Input } from "../../components/ui/input";

const sampleQuestions = [
  {
    id: 1,
    question: "Which option is the capital of France?",
    options: ["A. Paris", "B. Madrid", "C. Berlin", "D. Rome"],
    correct: "A",
  },
  {
    id: 2,
    question: "Which number is even?",
    options: ["A. 7", "B. 11", "C. 12", "D. 9"],
    correct: "C",
  },
  {
    id: 3,
    question: "What is 5 + 3?",
    options: ["A. 7", "B. 8", "C. 10", "D. 9"],
    correct: "B",
  },
];

export function StudentQuizPage() {
  const navigate = useNavigate();
  const token = localStorage.getItem("student_token");
  const studentCode = useMemo(
    () => localStorage.getItem("student_code") || "",
    [],
  );

  const [answers, setAnswers] = useState<Record<number, string>>({});
  const [submitted, setSubmitted] = useState(false);
  const [score, setScore] = useState<number | null>(null);

  const handleAnswer = (questionId: number, value: string) => {
    setAnswers((prev) => ({ ...prev, [questionId]: value }));
  };

  const handleSubmit = () => {
    const totalCorrect = sampleQuestions.reduce((count, question) => {
      return count + (answers[question.id] === question.correct ? 1 : 0);
    }, 0);

    setScore(totalCorrect);
    setSubmitted(true);
  };

  if (!token) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Unauthorized</CardTitle>
          <CardDescription>
            Please login with your student code before starting the test.
          </CardDescription>
          <CardContent>
            <Button onClick={() => navigate("/")}>Go to Login</Button>
          </CardContent>
        </CardHeader>
      </Card>
    );
  }

  return (
    <div className="flex w-full max-w-3xl flex-col gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Quiz Page</CardTitle>
          <CardDescription>
            Answer the questions below and submit when finished.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-slate-700">
            <p className="text-sm font-semibold">Student Code</p>
            <p>{studentCode}</p>
          </div>

          {sampleQuestions.map((question) => (
            <div
              key={question.id}
              className="space-y-3 rounded-xl border border-slate-200 p-4"
            >
              <p className="font-semibold">Question {question.id}</p>
              <p>{question.question}</p>
              <div className="grid gap-2 sm:grid-cols-2">
                {question.options.map((option) => {
                  const value = option[0];
                  return (
                    <label
                      key={option}
                      className="flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 hover:bg-slate-50"
                    >
                      <Input
                        type="radio"
                        name={`question-${question.id}`}
                        value={value}
                        checked={answers[question.id] === value}
                        onChange={() => handleAnswer(question.id, value)}
                      />
                      <span>{option}</span>
                    </label>
                  );
                })}
              </div>
            </div>
          ))}

          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <Button onClick={handleSubmit}>Submit Answers</Button>
            {submitted && (
              <div className="rounded-lg bg-slate-100 p-3 text-slate-800">
                <p className="font-semibold">Result</p>
                <p>
                  You answered {score} of {sampleQuestions.length} questions
                  correctly.
                </p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
