import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Container from '@mui/material/Container';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';

const appName = import.meta.env.VITE_APP_NAME;
const appTagline = import.meta.env.VITE_APP_TAGLINE;
const appDescription = import.meta.env.VITE_APP_DESCRIPTION;
const appVersion = import.meta.env.VITE_APP_VERSION;

export default function App() {
  return (
    <Container maxWidth="sm">
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          minHeight: '100vh',
          py: 4,
        }}
      >
        <Paper elevation={3} sx={{ p: 4, textAlign: 'center' }}>
          <Typography variant="h3" component="h1" gutterBottom>
            {appName}
          </Typography>
          <Typography variant="h6" color="text.secondary" gutterBottom>
            {appTagline}
          </Typography>
          <Typography variant="body1" sx={{ mt: 2 }}>
            {appDescription}
          </Typography>
          <Chip label={`v${appVersion}`} size="small" sx={{ mt: 3 }} />
        </Paper>
      </Box>
    </Container>
  );
}
