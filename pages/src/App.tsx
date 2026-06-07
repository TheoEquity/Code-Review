import React from 'react';
import { Routes, Route } from 'react-router-dom';
import AdminConsolePage from './pages/AdminConsolePage';
import DocsPage from './pages/DocsPage';

const App: React.FC = () => {
  return (
    <Routes>
      <Route path="/" element={<AdminConsolePage />} />
      <Route path="/admin" element={<AdminConsolePage />} />
      <Route path="/admin/repositories" element={<AdminConsolePage />} />
      <Route path="/admin/new-task" element={<AdminConsolePage />} />
      <Route path="/admin/task-history" element={<AdminConsolePage />} />
      <Route path="/admin/task-detail" element={<AdminConsolePage />} />
      <Route path="/admin/rules" element={<AdminConsolePage />} />
      <Route path="/admin/settings" element={<AdminConsolePage />} />
    </Routes>
  );
};

export default App;
