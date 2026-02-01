import { Router, Route } from '@solidjs/router';
import Chat from './pages/Chat';
import Composer from './pages/Composer';
import Observe from './pages/Observe';

export default function App() {
  return (
    <Router>
      <Route path="/" component={Chat} />
      <Route path="/composer" component={Composer} />
      <Route path="/observe" component={Observe} />
    </Router>
  );
}
