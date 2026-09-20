import { Link, Route, Routes, useLocation } from "react-router-dom";
import { AppProvider } from "./app/AppProvider";
import { Empty } from "./components/Feedback";
import { AdminPage } from "./features/admin/AdminPage";
import { HomePage } from "./features/feed/HomePage";
import { InfoPage } from "./features/info/InfoPage";
import { ModerationPage } from "./features/moderation/ModerationPage";
import { LoginPage } from "./features/staff/LoginPage";
import { NewTopicPage } from "./features/topics/NewTopicPage";
import { TopicPage } from "./features/topics/TopicPage";
import { AppLayout } from "./layouts/AppLayout";

export default function App() {
  const location = useLocation();
  return (
    <AppProvider>
      <AppLayout>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/new" element={<NewTopicPage />} />
          <Route
            path="/topics/:id"
            element={<TopicPage key={location.pathname} />}
          />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/moderation" element={<ModerationPage />} />
          <Route path="/admin" element={<AdminPage />} />
          <Route path="/rules" element={<InfoPage />} />
          <Route path="/about" element={<InfoPage about />} />
          <Route
            path="*"
            element={
              <Empty title="Такой страницы нет">
                <Link to="/">Вернуться к обсуждениям</Link>
              </Empty>
            }
          />
        </Routes>
      </AppLayout>
    </AppProvider>
  );
}
