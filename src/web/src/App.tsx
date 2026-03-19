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
import AlertDetailPage from "@/pages/alert-detail";
import PostDetailPage from "@/pages/post-detail";
import SourceDetailPage from "@/pages/source-detail";
import ConnectionDetailPage from "@/pages/connection-detail";
import ProcessingRuleDetailPage from "@/pages/processing-rule-detail";
import RuleDetailPage from "@/pages/rule-detail";
import KeywordDetailPage from "@/pages/keyword-detail";
import MemberDetailPage from "@/pages/member-detail";

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
          <Route path="alerts/:alertId" element={<AlertDetailPage />} />
          <Route path="keywords" element={<KeywordsPage />} />
          <Route path="keywords/:kwId" element={<KeywordDetailPage />} />
          <Route path="sources" element={<SourcesPage />} />
          <Route path="sources/:sourceId" element={<SourceDetailPage />} />
          <Route path="processing" element={<ProcessingPage />} />
          <Route path="processing/:ruleId" element={<ProcessingRuleDetailPage />} />
          <Route path="rules" element={<RulesPage />} />
          <Route path="rules/:ruleId" element={<RuleDetailPage />} />
          <Route path="connections" element={<ConnectionsPage />} />
          <Route path="connections/:connId" element={<ConnectionDetailPage />} />
          <Route path="posts" element={<PostsPage />} />
          <Route path="posts/:postId" element={<PostDetailPage />} />
          <Route path="members" element={<MembersPage />} />
          <Route path="members/:userId" element={<MemberDetailPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Route>
    </Routes>
  );
}
