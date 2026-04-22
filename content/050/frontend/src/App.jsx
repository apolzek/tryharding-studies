import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from './auth.jsx';
import Header from './components/Header.jsx';
import Login from './pages/Login.jsx';
import Register from './pages/Register.jsx';
import Home from './pages/Home.jsx';
import Profile from './pages/Profile.jsx';
import EditProfile from './pages/EditProfile.jsx';
import Friends from './pages/Friends.jsx';
import Communities from './pages/Communities.jsx';
import CommunityPage from './pages/CommunityPage.jsx';
import Scrapbook from './pages/Scrapbook.jsx';
import Testimonials from './pages/Testimonials.jsx';
import Photos from './pages/Photos.jsx';

function Protected({ children }) {
  const { user, loading } = useAuth();
  if (loading) return <div style={{ padding: 20 }}>carregando...</div>;
  if (!user) return <Navigate to="/login" replace />;
  return children;
}

export default function App() {
  const { user } = useAuth();
  return (
    <>
      {user && <Header />}
      <Routes>
        <Route path="/login" element={user ? <Navigate to="/home" /> : <Login />} />
        <Route path="/register" element={user ? <Navigate to="/home" /> : <Register />} />
        <Route path="/home" element={<Protected><Home /></Protected>} />
        <Route path="/profile/:id" element={<Protected><Profile /></Protected>} />
        <Route path="/profile/edit" element={<Protected><EditProfile /></Protected>} />
        <Route path="/friends" element={<Protected><Friends /></Protected>} />
        <Route path="/friends/:id" element={<Protected><Friends /></Protected>} />
        <Route path="/communities" element={<Protected><Communities /></Protected>} />
        <Route path="/community/:id" element={<Protected><CommunityPage /></Protected>} />
        <Route path="/scrapbook/:id" element={<Protected><Scrapbook /></Protected>} />
        <Route path="/testimonials/:id" element={<Protected><Testimonials /></Protected>} />
        <Route path="/photos/:id" element={<Protected><Photos /></Protected>} />
        <Route path="*" element={<Navigate to={user ? '/home' : '/login'} />} />
      </Routes>
    </>
  );
}
