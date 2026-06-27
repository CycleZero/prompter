import { useEffect, useState } from 'react';
import {
  List,
  ListItem,
  ListItemText,
  Typography,
  Paper,
  IconButton,
} from '@mui/material';
import { Delete, Refresh } from '@mui/icons-material';
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
    <Paper sx={{ p: 2 }}>
      <Typography variant="h6" gutterBottom>
        历史记录
        <IconButton onClick={fetch} size="small" sx={{ ml: 1 }}>
          <Refresh />
        </IconButton>
      </Typography>
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
    </Paper>
  );
}
