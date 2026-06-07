import { useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../../components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "../../components/ui/card";

export function StudentInstructionsPage() {
  const navigate = useNavigate();

  const studentCode = useMemo(
    () => localStorage.getItem("student_code") || "",
    [],
  );

  const handleAccept = () => {
    navigate("/quiz");
  };

  return (
    <div className="flex w-full max-w-3xl flex-col gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Student Test Instructions</CardTitle>
          <CardDescription>
            Please read the instructions carefully before starting your test.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-slate-700">
            <p className="text-sm font-semibold">Student Code</p>
            <p>{studentCode || "Not available"}</p>
          </div>

          <div className="space-y-2">
            <p className="font-semibold">Test Instructions</p>
            <ul className="list-disc space-y-2 pl-5 text-sm text-slate-700">
              <li>Write your answers by selecting the correct option.</li>
              <li>Each question is worth a fixed number of marks.</li>
              <li>Once you accept, you will be taken to the quiz page.</li>
              <li>Do not refresh the page during the test.</li>
              <li>If you are ready, click Accept and Begin.</li>
            </ul>
          </div>

          <div className="flex justify-end">
            <Button onClick={handleAccept}>Accept and Begin Test</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
