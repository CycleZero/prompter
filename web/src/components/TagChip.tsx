import { Box, Typography } from '@mui/material';

interface TagChipProps {
  /** 主要显示文本（中文翻译，显眼） */
  primary: string;
  /** 次要显示文本（英文原文，不太显眼） */
  secondary?: string;
  /** 是否选中（填充色） */
  selected?: boolean;
  onClick?: () => void;
  onDelete?: () => void;
}

/** 双行标签 Chip — 参考 WeiLin 风格，主显翻译、副显原文 */
export function TagChip({ primary, secondary, selected, onClick, onDelete }: TagChipProps) {
  return (
    <Box
      onClick={onClick}
      sx={{
        display: 'inline-flex',
        flexDirection: 'column',
        alignItems: 'center',
        px: 1.5,
        py: 0.5,
        borderRadius: 2,
        cursor: onClick ? 'pointer' : 'default',
        border: 1,
        borderColor: selected ? 'primary.main' : '#c5cae9',
        bgcolor: selected ? 'primary.main' : 'transparent',
        color: selected ? '#fff' : 'text.primary',
        '&:hover': onClick ? {
          bgcolor: selected ? 'primary.dark' : '#e3f2fd',
          borderColor: '#1976d2',
        } : {},
        position: 'relative',
        userSelect: 'none',
      }}
    >
      <Typography
        sx={{
          fontSize: '0.85rem',
          lineHeight: 1.3,
          fontWeight: selected ? 600 : 400,
        }}
      >
        {primary}
      </Typography>
      {secondary && (
        <Typography
          sx={{
            fontSize: '0.62rem',
            lineHeight: 1.2,
            color: selected ? 'rgba(255,255,255,0.7)' : 'text.secondary',
            maxWidth: 120,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {secondary}
        </Typography>
      )}
      {onDelete && (
        <Box
          component="span"
          onClick={(e) => { e.stopPropagation(); onDelete(); }}
          sx={{
            position: 'absolute',
            top: -4,
            right: -4,
            width: 16,
            height: 16,
            borderRadius: '50%',
            bgcolor: 'error.main',
            color: '#fff',
            fontSize: '0.6rem',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            cursor: 'pointer',
            lineHeight: 1,
          }}
        >
          ×
        </Box>
      )}
    </Box>
  );
}
