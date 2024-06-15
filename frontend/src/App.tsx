import './App.css'
import { Routes, Route } from 'react-router-dom';
import NavBar from './components/navbar'
import Homepage from './pages/Homepage'
import MoviePage from './pages/Movies';
import MovieDetail from './pages/MovieDetail';
import CredentialPage from './pages/CredentialPage'
import SeatSelectionPage from './pages/SeatSelectionPage';
function App() {
  return (
    <div className="w-full h-full bg-[#FDFCF0]">
      <NavBar />
      <Routes>
        <Route path="/login" element={<CredentialPage />} />
        <Route path="/movies" element={<MoviePage />} />
        <Route path="/movies/:imdbId" element={<MovieDetail />} />
        <Route path="/show/:showId" element={<SeatSelectionPage />} />
        <Route path="/" element={<Homepage />} />
      </Routes>
    </div>
  )
}
export default App
