import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { authAPI } from '../lib/api'
import { useAuth } from '../context/AuthContext'
import './Auth.css'

export default function SignUp() {
  const navigate = useNavigate()
  const { login } = useAuth()
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const result = await authAPI.signUp(formData.username, formData.email, formData.password)
      localStorage.setItem('token', result.token)
      login(result.token, result.user)
      
      const inFight = result.user?.inFight === true || result.user?.InFight === true
      if (inFight) {
        navigate('/fight', { replace: true })
      } else {
        navigate('/locations/moonshine', { replace: true })
      }
    } catch (err) {
      const errorMessage = err.message || ''
      const lowerMessage = errorMessage.toLowerCase()
      
      if (lowerMessage === 'user already exists') {
        setError('Пользователь с таким именем или email уже существует. Попробуйте другой.')
      } else if (lowerMessage === 'invalid input') {
        setError('Проверьте введенные данные. Имя пользователя и пароль должны содержать от 3 до 20 символов.')
      } else if (lowerMessage === 'invalid credentials') {
        setError('Неверные данные. Проверьте введенные данные.')
      } else {
        setError('Что-то пошло не так. Попробуйте позже.')
      }
    } finally {
      setLoading(false)
    }
  }

  const handleChange = (e) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    })
  }

  return (
    <div className="auth-container">
      <div className="auth-card">
        <h1>Регистрация</h1>
        <form onSubmit={handleSubmit}>
          {error && <div className="error-message">{error}</div>}
          <div className="form-group">
            <label htmlFor="username">Имя пользователя</label>
            <input
              type="text"
              id="username"
              name="username"
              value={formData.username}
              onChange={handleChange}
              required
              minLength={3}
              maxLength={20}
            />
          </div>
          <div className="form-group">
            <label htmlFor="email">Email</label>
            <input
              type="email"
              id="email"
              name="email"
              value={formData.email}
              onChange={handleChange}
              required
            />
          </div>
          <div className="form-group">
            <label htmlFor="password">Пароль</label>
            <input
              type="password"
              id="password"
              name="password"
              value={formData.password}
              onChange={handleChange}
              required
              minLength={3}
              maxLength={20}
            />
          </div>
          <button type="submit" disabled={loading}>
            {loading ? 'Регистрация...' : 'Зарегистрироваться'}
          </button>
        </form>
        <p className="auth-link">
          Уже есть аккаунт? <Link to="/signin">Войти</Link>
        </p>
      </div>
    </div>
  )
}
