# AI-Powered Student Diagnostic System - Frontend

This is the frontend of the AI-Powered Student Diagnostic System, built with **React**, **TypeScript**, and **Vite**.

## 📂 Project Structure

```text
src/
├── assets/             # Static assets (images, icons, etc.)
├── components/         # Shared reusable components
│   └── ui/             # Shadcn/ui core components
├── features/           # Feature-based modules (business logic + components)
│   ├── admin/          # Admin dashboard and management
│   ├── auth/           # Authentication related logic
│   ├── sqi/            # Student Quotient Index diagnostic
│   ├── student/        # Student dashboard and profile
│   └── test/           # General testing/diagnostic logic
├── lib/               # Third-party library configurations (e.g., axios, utils)
├── pages/             # Page components that map to routes
├── services/          # API services and integration logic
├── types/             # TypeScript interfaces and types
├── utils/             # Helper functions and formatting
├── App.tsx            # Main application component
├── App.css            # Global App-specific styles
├── index.css          # Tailwind and base global styles
└── main.tsx           # Application entry point
```

## 🛠️ Tech Stack

- **Framework:** React 18
- **Build Tool:** Vite
- **Language:** TypeScript
- **Styling:** Tailwind CSS + Shadcn/ui
- **State Management:** (Pending - e.g., TanStack Query / Redux)
- **Routing:** React Router DOM

## 🚀 Getting Started

### Prerequisites

- Node.js (v18 or higher)
- npm or yarn

### Installation

1. Navigate to the frontend directory:
   ```bash
   cd frontend
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Start the development server:
   ```bash
   npm run dev
   ```

4. Build for production:
   ```bash
   npm run build
   ```

## 📝 Features Under Development

- [ ] AI-Powered Student Diagnostic (SQI)
- [ ] Student & Admin Dashboards
- [ ] Real-time Progress Tracking
- [ ] Personalized Learning Paths
