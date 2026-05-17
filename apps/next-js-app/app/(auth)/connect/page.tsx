import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { joinUrl } from "@/lib/api";

export default function ConnectPage() {
  return (
    <Card className="w-full max-w-sm p-6">
      <CardHeader>
        <CardTitle>Sign in</CardTitle>
        <CardDescription>Continue with your Google account.</CardDescription>
      </CardHeader>
      <CardContent>
        <Button asChild className="w-full">
          <a href={joinUrl("/api/users/google")}>Sign in with Google</a>
        </Button>
      </CardContent>
    </Card>
  );
}
