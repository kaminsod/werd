import { Routes, Route, Navigate } from "react-router";
import ProtectedRoute from "@/components/protected-route";
import Layout from "@/components/layout";
import LoginPage from "@/pages/login";
import ProjectsPage from "@/pages/projects";
import AlertsPage from "@/pages/project-alerts";
import KeywordsPage from "@/pages/project-keywords";
import SourcesPage from "@/pages/project-sources";
import RulesPage from "@/pages/project-rules";
import ConnectionsPage from "@/pages/project-connections";
import PostsPage from "@/pages/project-posts";
import MembersPage from "@/pages/project-members";
import ProcessingPage from "@/pages/project-processing";
import SettingsPage from "@/pages/project-settings";

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<ProtectedRoute />}>
        <Route path="/" element={<Navigate to="/projects" replace />} />
        <Route path="/projects" element={<ProjectsPage />} />
        <Route path="/projects/:id" element={<Layout />}>
          <Route index element={<Navigate to="alerts" replace />} />
          <Route path="alerts" element={<AlertsPage />} />
          <Route path="keywords" element={<KeywordsPage />} />
          <Route path="sources" element={<SourcesPage />} />
          <Route path="processing" element={<ProcessingPage />} />
          <Route path="rules" element={<RulesPage />} />
          <Route path="connections" element={<ConnectionsPage />} />
          <Route path="posts" element={<PostsPage />} />
          <Route path="members" element={<MembersPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Route>
    </Routes>
  );
}
