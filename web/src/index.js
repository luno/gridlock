import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import '@fontsource/source-sans-pro/index.css'
import App from './App';
import reportWebVitals from './reportWebVitals';

import 'bootstrap';
import 'bootstrap/dist/css/bootstrap.css';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { createBrowserHistory } from 'history';

import WebFont from 'webfontloader';

export const history = createBrowserHistory({
  basename: process.env.PUBLIC_URL
});

function fontsActive () {
  ReactDOM.render(
      <BrowserRouter history={history} basename={'gridlock'}>
          <Routes>
              <Route path="/" element={<App />}></Route>
          </Routes>
      </BrowserRouter>,
    document.getElementById('root')
  );
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
