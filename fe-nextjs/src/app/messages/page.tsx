import { BottomNav } from "@/features/catalog/bottom-nav";
import { Header } from "@/features/catalog/header";
import { MessagesPageContent } from "@/features/chat/messages-page-content";

export default function MessagesPage() {
  return (
    <>
      <Header />

      <div className="flex flex-1 flex-col">
        <MessagesPageContent />
      </div>

      <BottomNav />
    </>
  );
}
