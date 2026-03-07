import { Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import SignUp from './pages/SignUp'
import SignIn from './pages/SignIn'
import Location from './pages/Location'
import Profile from './pages/Profile'
import EquipmentItems from './pages/EquipmentItems'
import ToolItemsShop from './pages/ToolItemsShop'
import Bots from './pages/Bots'
import Fight from './pages/Fight'
import ResourcesGather from './pages/ResourcesGather'
import ProtectedRoute from './components/ProtectedRoute'

function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/signup" element={<SignUp />} />
        <Route path="/signin" element={<SignIn />} />
        <Route
          path="/locations/:slug"
          element={
            <ProtectedRoute>
              <Location />
            </ProtectedRoute>
          }
        />
        <Route
          path="/profile"
          element={
            <ProtectedRoute>
              <Profile />
            </ProtectedRoute>
          }
        />
        <Route
          path="/equipment_items"
          element={
            <ProtectedRoute>
              <EquipmentItems />
            </ProtectedRoute>
          }
        />
        <Route
          path="/tool_items/shop"
          element={
            <ProtectedRoute>
              <ToolItemsShop />
            </ProtectedRoute>
          }
        />
        <Route
          path="/bots/:location_slug"
          element={
            <ProtectedRoute>
              <Bots />
            </ProtectedRoute>
          }
        />
        <Route
          path="/fight"
          element={
            <ProtectedRoute>
              <Fight />
            </ProtectedRoute>
          }
        />
        <Route
          path="/resources/:location_slug"
          element={
            <ProtectedRoute>
              <ResourcesGather />
            </ProtectedRoute>
          }
        />
        <Route
          path="/"
          element={<Navigate to="/locations/moonshine" replace />}
        />
        <Route path="*" element={<Navigate to="/signin" replace />} />
      </Routes>
    </AuthProvider>
  )
}

export default App
