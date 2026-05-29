import React from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import { useHealthStore } from '@/api';
import { LoginButton } from './LoginButton';

export const NavBar: React.FC = () => {
  const { status } = useHealthStore();
  const navigate = useNavigate();

  const handleHealthClick = () => {
    navigate('/health');
  };

  const getHealthStyles = () => {
    switch (status) {
      case 'healthy':
        return {
          pillClass: 'health-pill-healthy',
          dotClass: 'health-dot-healthy',
          text: 'RUNTIME ONLINE',
        };
      case 'unreachable':
        return {
          pillClass: 'health-pill-unreachable',
          dotClass: 'health-dot-unreachable',
          text: 'RUNTIME UNREACHABLE',
        };
      case 'loading':
      default:
        return {
          pillClass: 'health-pill-loading',
          dotClass: 'health-dot-loading',
          text: 'RUNTIME CONNECTING',
        };
    }
  };

  const health = getHealthStyles();

  return (
    <nav className="sentinel-navbar" id="app-navbar">
      <div className="navbar-left">
        <NavLink to="/" className="navbar-logo" id="nav-logo">
          <span className="logo-text">Agent<span className="logo-accent">Verse</span></span>
          <span className="logo-v1">v1</span>
        </NavLink>
        <div className="navbar-links" id="nav-links">
          <NavLink to="/" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} end id="nav-link-canvas">
            Canvas
          </NavLink>
          <NavLink to="/dashboard" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} id="nav-link-dashboard">
            Dashboard
          </NavLink>
          <NavLink to="/agent-studio" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} id="nav-link-studio">
            Agent Studio
          </NavLink>
          <NavLink to="/flows" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} id="nav-link-flows">
            Flows
          </NavLink>
          <NavLink to="/finops" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} id="nav-link-finops">
            FinOps
          </NavLink>
          <NavLink to="/memory" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} id="nav-link-memory">
            Memory
          </NavLink>
        </div>
      </div>
      <div className="navbar-right">
        <button
          className={`health-pill ${health.pillClass}`}
          onClick={handleHealthClick}
          title="Click to view detailed system health status"
          id="cao-health-pill"
        >
          <span className={`health-dot ${health.dotClass}`}></span>
          <span className="health-text">{health.text}</span>
        </button>
        <LoginButton />
      </div>
    </nav>
  );
};
