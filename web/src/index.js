import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import '@fontsource/source-sans-pro/index.css'
import App from './App';
import reportWebVitals from './reportWebVitals';

import 'bootstrap';
import 'bootstrap/dist/css/bootstrap.css';

import WebFont from 'webfontloader';

function fontsActive () {
  ReactDOM.render(
    <App />,
    document.getElementById('root')
  );
  // ReactDOM.render(
  //   <React.StrictMode>
  //     <App />
  //   </React.StrictMode>,
  //   document.getElementById('root')
  // );
}

WebFont.load({
  custom: {
    families: ['Source Sans Pro:n3,n4,n6,n7']
  },
  active: fontsActive,
});


// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
