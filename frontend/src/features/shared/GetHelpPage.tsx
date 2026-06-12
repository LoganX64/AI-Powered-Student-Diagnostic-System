import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import {
  CircleHelpIcon,
  MessageSquareIcon,
  BookOpenIcon,
  ExternalLinkIcon,
  CheckCircle2Icon,
  CircleDotIcon,
  HelpCircleIcon,
} from "lucide-react";
import { mockFaqs } from "@/features/shared/mockData";

export function GetHelpPage() {
  const [contactForm, setContactForm] = useState({
    subject: "",
    message: "",
  });
  const [submitting, setSubmitting] = useState(false);

  const handleContactSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    await new Promise((r) => setTimeout(r, 1000));
    toast.success("Your message has been sent. We'll get back to you soon!");
    setContactForm({ subject: "", message: "" });
    setSubmitting(false);
  };

  return (
    <Tabs defaultValue="faq" className="w-full">
      <TabsList className="grid w-full grid-cols-3">
        <TabsTrigger value="faq" className="flex items-center gap-2">
          <CircleHelpIcon className="size-4" />
          FAQ
        </TabsTrigger>
        <TabsTrigger value="contact" className="flex items-center gap-2">
          <MessageSquareIcon className="size-4" />
          Contact Support
        </TabsTrigger>
        <TabsTrigger value="docs" className="flex items-center gap-2">
          <BookOpenIcon className="size-4" />
          Documentation
        </TabsTrigger>
      </TabsList>

      <TabsContent value="faq" className="mt-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CircleHelpIcon className="size-5" />
              Frequently Asked Questions
            </CardTitle>
            <CardDescription>
              Find answers to common questions about the platform.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            {mockFaqs.map((faq) => (
              <div key={faq.id} className="flex flex-col gap-2 p-4 rounded-lg border">
                <div className="flex items-start gap-2">
                  <HelpCircleIcon className="size-5 text-primary shrink-0 mt-0.5" />
                  <div>
                    <h3 className="font-medium">{faq.question}</h3>
                    <p className="text-sm text-muted-foreground mt-1">{faq.answer}</p>
                  </div>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="contact" className="mt-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MessageSquareIcon className="size-5" />
              Contact Support
            </CardTitle>
            <CardDescription>
              Send us a message and we'll respond within 24 hours.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleContactSubmit} className="flex flex-col gap-4 max-w-lg">
              <div className="flex flex-col gap-2">
                <Label htmlFor="subject">Subject</Label>
                <Input
                  id="subject"
                  placeholder="How can we help?"
                  value={contactForm.subject}
                  onChange={(e) =>
                    setContactForm({ ...contactForm, subject: e.target.value })
                  }
                  required
                />
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="message">Message</Label>
                <Textarea
                  id="message"
                  placeholder="Describe your issue or question in detail..."
                  rows={5}
                  value={contactForm.message}
                  onChange={(e) =>
                    setContactForm({ ...contactForm, message: e.target.value })
                  }
                  required
                />
              </div>
              <Button type="submit" disabled={submitting} className="w-fit">
                {submitting ? "Sending..." : "Send Message"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="docs" className="mt-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BookOpenIcon className="size-5" />
              Documentation
            </CardTitle>
            <CardDescription>
              Explore our guides and documentation.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <div className="grid gap-4 md:grid-cols-2">
              <a
                href="#"
                className="flex items-center justify-between p-4 rounded-lg border hover:bg-accent transition-colors"
              >
                <div className="flex items-center gap-3">
                  <CheckCircle2Icon className="size-5 text-green-500" />
                  <div>
                    <h3 className="font-medium">Getting Started</h3>
                    <p className="text-sm text-muted-foreground">
                      Learn the basics of the platform
                    </p>
                  </div>
                </div>
                <ExternalLinkIcon className="size-4 text-muted-foreground" />
              </a>
              <a
                href="#"
                className="flex items-center justify-between p-4 rounded-lg border hover:bg-accent transition-colors"
              >
                <div className="flex items-center gap-3">
                  <CircleDotIcon className="size-5 text-blue-500" />
                  <div>
                    <h3 className="font-medium">Test Management</h3>
                    <p className="text-sm text-muted-foreground">
                      Create and manage assessments
                    </p>
                  </div>
                </div>
                <ExternalLinkIcon className="size-4 text-muted-foreground" />
              </a>
              <a
                href="#"
                className="flex items-center justify-between p-4 rounded-lg border hover:bg-accent transition-colors"
              >
                <div className="flex items-center gap-3">
                  <Badge variant="secondary">API</Badge>
                  <div>
                    <h3 className="font-medium">API Reference</h3>
                    <p className="text-sm text-muted-foreground">
                      Integrate with our API
                    </p>
                  </div>
                </div>
                <ExternalLinkIcon className="size-4 text-muted-foreground" />
              </a>
              <a
                href="#"
                className="flex items-center justify-between p-4 rounded-lg border hover:bg-accent transition-colors"
              >
                <div className="flex items-center gap-3">
                  <Badge variant="outline">Video</Badge>
                  <div>
                    <h3 className="font-medium">Video Tutorials</h3>
                    <p className="text-sm text-muted-foreground">
                      Watch step-by-step guides
                    </p>
                  </div>
                </div>
                <ExternalLinkIcon className="size-4 text-muted-foreground" />
              </a>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  );
}
