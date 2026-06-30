import { useEffect, useState } from 'react';
import {
  Box,
  List,
  ListItem,
  ListItemText,
  Typography,
  Paper,
  IconButton,
  Button,
} from '@mui/material';
import { Delete, Refresh, ArrowBack } from '@mui/icons-material';
import { api } from '../api/client';
import type { Record } from '../types';

/** 历史记录页面 — 展示已持久化的 Prompt 记录列表 */
export function RecordsPage() {
  const [records, setRecords] = useState<Record[]>([]);

  /** 获取记录列表 */
  const fetch = () =>
    api.listRecords(1, 20).then((res) => setRecords(res.data.list));

  useEffect(() => {
    fetch();
  }, []);

  /** 删除指定记录 */
  const handleDelete = async (id: number) => {
    await api.deleteRecord(id);
    fetch();
  };

  return (
    <Box sx={{ p: 2, flexGrow: 1, overflow: 'auto' }}>
      {/* 返回按钮 */}
      <Button
        startIcon={<ArrowBack />}
        onClick={() => { window.location.hash = ''; }}
        size="small"
        sx={{ mb: 1 }}
      >
        返回编辑器
      </Button>

      <Paper sx={{ p: 2 }} variant="outlined">
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
          <Typography variant="h6" sx={{ flexGrow: 1 }}>
            历史记录
          </Typography>
          <IconButton onClick={fetch} size="small">
            <Refresh />
          </IconButton>
        </Box>
        {records.length === 0 ? (
          <Typography variant="body2" color="text.secondary">
            暂无保存记录
          </Typography>
        ) : (
          <List>
            {records.map((r) => (
              <ListItem
                key={r.id}
                secondaryAction={
                  <IconButton
                    edge="end"
                    onClick={() => handleDelete(r.id)}
                  >
                    <Delete />
                  </IconButton>
                }
              >
                <ListItemText
                  primary={r.title || '(无标题)'}
                  secondary={r.full_content?.slice(0, 80) + '...'}
                />
              </ListItem>
            ))}
          </List>
        )}
      </Paper>
    </Box>
  );
}
