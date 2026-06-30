import { useState, useEffect } from 'react';
import {
  AppBar,
  Toolbar,
  Typography,
  Button,
  Box,
  CssBaseline,
  ThemeProvider,
  createTheme,
} from '@mui/material';
import { EditorPage } from './pages/EditorPage';
import { RecordsPage } from './pages/RecordsPage';

// 浅色主题 — 默认使用明亮配色，参考 WeiLin + Cloudreve 现代风格
const theme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#1976d2' },
    background: { default: '#f5f7fa', paper: '#ffffff' },
  },
  shape: { borderRadius: 8 },
  components: {
    MuiChip: {
      styleOverrides: {
        outlined: {
          borderColor: '#c5cae9',
          '&:hover': { backgroundColor: '#e3f2fd' },
        },
      },
    },
  },
});

function App() {
  const [page, setPage] = useState<'editor' | 'records'>('editor');

  // 监听 hash 变化，实现简单的页面切换（MVP 阶段无需路由库）
  useEffect(() => {
    const handleHashChange = () => {
      setPage(window.location.hash === '#records' ? 'records' : 'editor');
    };
    handleHashChange();
    window.addEventListener('hashchange', handleHashChange);
    return () => window.removeEventListener('hashchange', handleHashChange);
  }, []);

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
        {/* 顶部导航栏 */}
        <AppBar position="static" color="inherit" elevation={0} sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Toolbar>
            <Typography variant="h6" sx={{ fontWeight: 700, color: 'primary.main', flexGrow: 1 }}>
              🔵 Prompter
            </Typography>
            <Button
              variant={page === 'editor' ? 'contained' : 'outlined'}
              size="small"
              onClick={() => { window.location.hash = ''; }}
              sx={{ mr: 1 }}
            >
              编辑器
            </Button>
            <Button
              variant={page === 'records' ? 'contained' : 'outlined'}
              size="small"
              onClick={() => { window.location.hash = 'records'; }}
            >
              记录
            </Button>
          </Toolbar>
        </AppBar>

        {/* 页面内容 */}
        {page === 'editor' ? <EditorPage /> : <RecordsPage />}
      </Box>
    </ThemeProvider>
  );
}

export default App;
