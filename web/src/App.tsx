import {
  BrowserRouter,
  Routes,
  Route,
  Link as RouterLink,
} from 'react-router-dom';
import {
  AppBar,
  Toolbar,
  Typography,
  Button,
  Container,
  CssBaseline,
  ThemeProvider,
  createTheme,
} from '@mui/material';
import { EditorPage } from './pages/EditorPage';
import { RecordsPage } from './pages/RecordsPage';

// 全局深色主题
const theme = createTheme({
  palette: { mode: 'dark' },
});

function App() {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <BrowserRouter>
        {/* 顶部导航栏 */}
        <AppBar position="sticky">
          <Toolbar>
            <Typography variant="h6" sx={{ flexGrow: 1 }}>
              Prompter
            </Typography>
            <Button color="inherit" component={RouterLink} to="/">
              编辑器
            </Button>
            <Button color="inherit" component={RouterLink} to="/records">
              记录
            </Button>
          </Toolbar>
        </AppBar>

        {/* 页面容器 */}
        <Container maxWidth="xl" sx={{ mt: 2 }}>
          <Routes>
            <Route path="/" element={<EditorPage />} />
            <Route path="/records" element={<RecordsPage />} />
          </Routes>
        </Container>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;
